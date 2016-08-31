package main

import (
  "errors"
  "syscall"
  "strconv"
  "fmt"
  "io"
  "os"
  "time"

  ma "github.com/jbenet/go-multiaddr"
  manet "github.com/jbenet/go-multiaddr-net"
)

var usageText = `usage: %s <fd-num> <multiaddr>

open <fd-num> file descriptors at <multiaddr>, and keep them open
until the process is killed. This command useful for test suites.
For example:

    # open 16 tcp sockets at 127.0.0.1:80
    %s 16 /ip4/127.0.0.1/tcp/80
`
// TODO:
// # open 64 utp sockets at 127.0.0.1:8080
// %s 64 /ip4/127.0.0.1/udp/8080/utp

// # open 1024 unix domain sockets at /foo/var.sock
// %s 1024 /uds/foo/var.sock

func fdRaise(nn int) error {
  n := uint64(nn)

  var rLimit syscall.Rlimit
  err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
  if err != nil {
    return err
  }

  if rLimit.Cur >= n {
    fmt.Printf("already at %d >= %d fds\n", rLimit.Cur, n)
    return nil // all good.
  }

  err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
  if err != nil {
    return err
  }

  err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
  if err != nil {
    return err
  }

  if rLimit.Cur < n {
    return fmt.Errorf("failed to raise fd limit to %d (still %d)", n, rLimit.Cur)
  }


  fmt.Printf("raised fds to %d >= %d fds\n", rLimit.Cur, n)
  return nil
}

func readUntilErr(r io.Reader) error {
  buf := make([]byte, 1024)
  for {
    _, err := r.Read(buf)
    if err != nil {
      return err
    }
  }
}

func dialAndHang(i int, m ma.Multiaddr, errs chan<- error) {
  c, err := manet.Dial(m)
  if err != nil {
    errs<- err
    return
  }

  fmt.Printf("conn %d connected\n", i)
  // read until proc exits or conn closes.
  errs<- readUntilErr(c)
}

func fdHang(n int, m ma.Multiaddr) error {
  // first, make sure we raise our own fds to be enough.
  if err := fdRaise(n + 10); err != nil {
    return err
  }

  errs := make(chan error, n)
  for i := 0; i < n; i++ {
    i := i

    // this sleep is here because OSX fails to be able to dial and listen
    // as fast as Go tries to issue the commands. this seems to be a crap
    // os failure.
    time.Sleep(time.Millisecond)

    go dialAndHang(i, m, errs)
  }

  for i := 0; i < n; i++ {
    err := <-errs
    if err != nil && err != io.EOF {
      fmt.Printf("conn %d error: %s\n", i, err)
    }
  }

  fmt.Println("done")
  return nil
}

func run(args []string) error {
  n, err := strconv.Atoi(args[0])
  if err != nil {
    return errors.New("<fd-num> argument must be a number")
  }

  m, err := ma.NewMultiaddr(args[1])
  if err != nil {
    return errors.New("<multiaddr> argument must be a valid multiaddr")
  }

  fmt.Printf("hanging %d fds at %s\n", n, m)
  return fdHang(n, m)
}

func main() {
  usageAndExit := func(code int) {
    p := os.Args[0]
    fmt.Printf(usageText, p, p)
    os.Exit(code)
  }

  for _, arg := range os.Args {
    if arg == "-h" || arg == "--help" {
      usageAndExit(0)
      return
    }
  }
  if len(os.Args) != 3 {
    usageAndExit(-1)
    return
  }

  if err := run(os.Args[1:]); err != nil {
    fmt.Fprintf(os.Stderr, "error: %s\n", err)
    os.Exit(-1)
  }
}
