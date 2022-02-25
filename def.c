/*
helpful links and information that
will help with further development.

issues:
https://github.com/golang/go/issues/20639 <- using //extern|//export funcToCFromGo
	https://golang.org/cmd/cgo/#hdr-C_references_to_Go
https://www.cprogramming.com/declare_vs_define.html <- trying to get around ^^ fuckery
https://www.tutorialspoint.com/cprogramming/c_functions.htm <- again trying to get around the fuckery above
https://man7.org/linux/man-pages/man2/sigaction.2.html <- manpage for C signal handling that I need to implement
https://stackoverflow.com/questions/1716296/why-does-printf-not-flush-after-the-call-unless-a-newline-is-in-the-format-strin
	^^ stupid C flush printing stuff. Fix: C.setbuf(C.stdout, nil)
https://stackoverflow.com/questions/572547/what-does-static-mean-in-c <- study this

guides;
https://golang.org/cmd/cgo <- main documentation
https://golang.org/doc/articles/c_go_cgo.html <- main cgo release document
https://documentation.help/Golang/cgo.html < cgo basics
https://people.kth.se/~johanmon/ose/assignments/signals.pdf <- nice pdf about C signals
*/

/* from https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go:
Using //export in a file places a restriction on the preamble: since it is copied into two different C output files, it must not contain any definitions, only declarations. If a file contains both definitions and declarations, then the two output files will produce duplicate symbols and the linker will fail. To avoid this, definitions must be placed in preambles in other files, or in C source files.
*/
#include <signal.h>

/*
extern void sigHandler(int signum, siginfo_t *si, void *udcontext);
extern struct sigaction sa;
extern void addSig(int sig);
*/

//void sigHandler(int signum, siginfo_t *si, void *ucontext);
// struct sigaction sa;
//void addSig(int sig);
