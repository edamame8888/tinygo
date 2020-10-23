// +build darwin linux,!baremetal,!wasi freebsd,!baremetal
// +build !nintendoswitch

package runtime

import (
	"unsafe"
)

//export putchar
func _putchar(c int) int

//export strlen
func strlen(s *byte) uintptr

//export usleep
func usleep(usec uint) int

//export malloc
func malloc(size uintptr) unsafe.Pointer

//export abort
func abort()

//export exit
func exit(code int)

//export clock_gettime
func clock_gettime(clk_id int32, ts *timespec)

type timeUnit int64

// Note: tv_sec and tv_nsec vary in size by platform. They are 32-bit on 32-bit
// systems and 64-bit on 64-bit systems (at least on macOS/Linux), so we can
// simply use the 'int' type which does the same.
type timespec struct {
	tv_sec  int // time_t: follows the platform bitness
	tv_nsec int // long: on Linux and macOS, follows the platform bitness
}

const CLOCK_MONOTONIC_RAW = 4

var stackTop uintptr

func postinit() {}

// Entry point for Go. Initialize all packages and call main.main().
//export main
func main(argc int32, argv **byte) int {
	preinit()

	// Make args global big enough so that it can store all command line
	// arguments. Unfortunately this has to be done with some magic as the heap
	// is not yet initialized.
	argsSlice := (*struct {
		ptr unsafe.Pointer
		len uintptr
		cap uintptr
	})(unsafe.Pointer(&args))
	argsSlice.ptr = malloc(uintptr(argc) * (unsafe.Sizeof(uintptr(0))) * 3)
	argsSlice.len = uintptr(argc)
	argsSlice.cap = uintptr(argc)

	// Initialize command line parameters. Again, using some magic, this time to
	// convert (argc, argv) to a Go slice without doing any memory allocations.
	argvSlice := (*[1 << 16]*byte)(unsafe.Pointer(argv))[:argc]
	for i, ptr := range argvSlice {
		argString := (*_string)(unsafe.Pointer(&args[i]))
		argString.length = strlen(ptr)
		argString.ptr = ptr
	}

	// Obtain the initial stack pointer right before calling the run() function.
	// The run function has been moved to a separate (non-inlined) function so
	// that the correct stack pointer is read.
	stackTop = getCurrentStackPointer()
	runMain()

	// For libc compatibility.
	return 0
}

// Must be a separate function to get the correct stack pointer.
//go:noinline
func runMain() {
	run()
}

func putchar(c byte) {
	_putchar(int(c))
}

const asyncScheduler = false

func ticksToNanoseconds(ticks timeUnit) int64 {
	// The OS API works in nanoseconds so no conversion necessary.
	return int64(ticks)
}

func nanosecondsToTicks(ns int64) timeUnit {
	// The OS API works in nanoseconds so no conversion necessary.
	return timeUnit(ns)
}

func sleepTicks(d timeUnit) {
	// timeUnit is in nanoseconds, so need to convert to microseconds here.
	usleep(uint(d) / 1000)
}

// Return monotonic time in nanoseconds.
//
// TODO: noescape
func monotime() uint64 {
	ts := timespec{}
	clock_gettime(CLOCK_MONOTONIC_RAW, &ts)
	return uint64(ts.tv_sec)*1000*1000*1000 + uint64(ts.tv_nsec)
}

func ticks() timeUnit {
	return timeUnit(monotime())
}

//go:linkname syscall_Exit syscall.Exit
func syscall_Exit(code int) {
	exit(code)
}

func extalloc(size uintptr) unsafe.Pointer {
	return malloc(size)
}

//export free
func extfree(ptr unsafe.Pointer)

// TinyGo does not yet support any form of parallelism on an OS, so these can be
// left empty.

//go:linkname procPin sync/atomic.runtime_procPin
func procPin() {
}

//go:linkname procUnpin sync/atomic.runtime_procUnpin
func procUnpin() {
}
