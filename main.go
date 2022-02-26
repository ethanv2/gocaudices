package main

/*
#include <stdio.h>
#include <signal.h>
#include "def.c"

extern void runFromC(int, int);

static void sigHandler(int signum, siginfo_t *si, void *ucontext) {
	printf("%d signal recieved\n", signum);
	int signal = signum - SIGRTMIN;
	printf("10\n");
	int button = si->si_value.sival_int;
	printf("20\n");
	runFromC(signal, button);
	printf("30\n");
}

static void addSig(int sig) {
	printf("added sig %d\n", sig);
	// static struct sigaction sa = { .sa_sigaction = sigHandler, .sa_flags = SA_ONSTACK };
	// printf("1\n");
	static struct sigaction sa = { .sa_sigaction = sigHandler, .sa_flags = SA_SIGINFO };
	int e = sigaction(sig, &sa, NULL);
	// printf("2\n");
	if(e != 0)
	printf("Failed to add signal: %d\n", sig);
	// printf("3\n");
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type block struct {
	cmd   string
	inSh  bool
	upInt int
	upSig int

	args []string
	pos  int
}

var updateChan = make(chan struct{}, len(blocks))
var barBytesArr = make([][]byte, len(blocks))
var signalMap = make(map[os.Signal][]block)

func (b *block) run() {
	outputBytes, err := exec.Command(b.args[0], b.args[1:]...).Output()
	if err != nil {
		log.Printf("Failed to update `%s`: %s\n", b.cmd, err)
		return
	}

	barBytesArr[b.pos] = bytes.TrimSpace(bytes.Split(outputBytes, []byte("\n"))[0])
	updateChan <- struct{}{}
}

func (b *block) runBB(button int) {
	cmd := exec.Command(b.args[0], b.args[1:]...)
	cmd.Env = append(cmd.Env, "BLOCK_BUTTON="+strconv.Itoa(button))
	outputBytes, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to update `%s`: %s\n", b.cmd, err)
		return
	}

	barBytesArr[b.pos] = bytes.TrimSpace(bytes.Split(outputBytes, []byte("\n"))[0])
	updateChan <- struct{}{}
}

//export runFromC
func runFromC(signal, button C.int) {
	// crashes while trying to running this
	fmt.Printf("signal: %d; button: %d\n", signal, button)
	sig, butt := int(signal), int(button)
	for _, b := range signalMap[syscall.Signal(sig)] {
		if butt == 0 {
			b.run()
		} else {
			b.runBB(butt)
		}
	}
}

func main() {
	C.setbuf(C.stdout, nil)

	x, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Cannot connect to X: %s\n", err)
	}
	defer x.Close()
	root := xproto.Setup(x).DefaultScreen(x).Root

	fmt.Printf("%d - %d\n\n", 0, len(blocks))
	for i := 0; i < len(blocks); i++ {
		// fmt.Printf("initializing block %+v\n", blocks[i])
		blocks[i].pos = i

		if blocks[i].inSh {
			blocks[i].args = []string{shell, cmdstropt, blocks[i].cmd}
		} else {
			blocks[i].args = strings.Split(blocks[i].cmd, " ")
		}

		if blocks[i].upSig != 0 {
			fmt.Printf("Block `%s` sig `%d`->`%d` added using `C.addSig()`\n", blocks[i].cmd, blocks[i].upSig, blocks[i].upSig+34)
			if _, err := C.addSig(C.int(34 + blocks[i].upSig)); err != nil {
				fmt.Printf("Failed adding signal `%d`->`%d`: %s\n", blocks[i].upSig, blocks[i].upSig+34, err)
			}
			// fmt.Printf("01\n")
			signalMap[syscall.Signal(34+blocks[i].upSig)] = append(signalMap[syscall.Signal(34+blocks[i].upSig)], blocks[i])
			//fmt.Printf("02\n")
		}

		blocks[i].run()
		// fmt.Printf("03\n")
		if blocks[i].upInt != 0 {
			go func(i int) {
				for {
					time.Sleep(time.Duration(blocks[i].upInt) * time.Second)
					blocks[i].run()
				}
			}(i)
		}
		// fmt.Printf("Finished initializing `%s`\n", blocks[i].cmd)
	}

	go func() {
		var finalBytesBuffer bytes.Buffer
		for range updateChan {
			for i := 0; i < len(blocks); i++ {
				if barBytesArr[i] != nil {
					finalBytesBuffer.Write(delim)
					finalBytesBuffer.Write(barBytesArr[i])
				}
			}

			finalBytes := bytes.TrimPrefix(finalBytesBuffer.Bytes(), delim)
			xproto.ChangeProperty(x, xproto.PropModeReplace, root, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(finalBytes)), finalBytes) // set the root window name
			finalBytesBuffer.Reset()
		}
	}()

	select {}
}
