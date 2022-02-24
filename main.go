package main

// #include <signal.h>
// #include <stdio.h>
//
// extern void runFromC(int, int);
//
// static void sighandler(int signum, siginfo_t *si, void *ucontext)
// {
//	printf("%d signal recieved\n", signum);
//	int signal = signum - SIGRTMIN;
//	int button = si->si_value.sival_int;
//	runFromC(signal, button);
// }
//
// static void addSig(int sig) {
// 	struct sigaction sa = { .sa_sigaction = sighandler, .sa_flags = SA_SIGINFO|SA_ONSTACK };
//	sigaction(sig, &sa, NULL);
// }
import "C"

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	// "os/signal"
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

var updateChan = make(chan struct{})
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
	x, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Cannot connect to X: %s\n", err)
	}
	defer x.Close()
	root := xproto.Setup(x).DefaultScreen(x).Root

	for i := 0; i < len(blocks); i++ {
		blocks[i].pos = i

		if blocks[i].inSh {
			blocks[i].args = []string{shell, cmdstropt, blocks[i].cmd}
		} else {
			blocks[i].args = strings.Split(blocks[i].cmd, " ")
		}

		if blocks[i].upSig != 0 {
			C.addSig(C.int(34 + blocks[i].upSig))
			signalMap[syscall.Signal(34+blocks[i].upSig)] = append(signalMap[syscall.Signal(34+blocks[i].upSig)], blocks[i])
		}

		blocks[i].run()
		if blocks[i].upInt != 0 {
			go func(i int) {
				for {
					time.Sleep(time.Duration(blocks[i].upInt) * time.Second)
					blocks[i].run()
				}
			}(i)
		}
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
