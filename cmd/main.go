package main

//TODO: AXFR
//TODO: netrange

import (
  "bufio"
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "runtime"
  "strings"
  "sync"
  "time"

  //"runtime/debug"
  "runtime/pprof"

  "github.com/camelinc/dnsDisco/shared"
  "github.com/op/go-logging"
)

var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
  `%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

const (
  TIMEOUT time.Duration = 3 // seconds
)

//distributor
// recv result in channel
// send work via channel
// send stop
//func dispatcher(taskChan chan Task) resultChan chan Result{

//worker
// recv work via channel
//   recvCh := make(chan string, 5)
// recv stop
// send result via channel
// func worker(taskChan chan Task) resultChan chan Eesult{

func setupLogging(log *logging.Logger, debugOutput *bool) {
  // For demo purposes, create two backend for os.Stderr.
  backend2 := logging.NewLogBackend(os.Stderr, "", 0)

  // For messages written to backend2 we want to add some additional
  // information to the output, including the used log level and the name of
  // the function.
  backend2Formatter := logging.NewBackendFormatter(backend2, format)
  backend2FormatterLeveled := logging.AddModuleLevel(backend2Formatter)
  if *debugOutput {
    backend2FormatterLeveled.SetLevel(logging.DEBUG, "")
  } else {
    backend2FormatterLeveled.SetLevel(logging.INFO, "")
  }

  logging.SetBackend(backend2FormatterLeveled)

  // Set the backends to be used.
  var logResults = logging.MustGetLogger("results")
  var format3 = logging.MustStringFormatter(`%{message}`)
  backend3 := logging.NewLogBackend(os.Stdout, "", 0)
  backend3Formatter := logging.NewBackendFormatter(backend3, format3)
  backend3FormatterLeveled := logging.AddModuleLevel(backend3Formatter)
  backend3FormatterLeveled.SetLevel(logging.INFO, "")

  logResults.SetBackend(backend3FormatterLeveled)

  return
  log.Debugf("debug %s", "test")
  log.Infof("info")
  log.Notice("notice")
  log.Warning("warning")
  log.Error("err")
  log.Critical("crit")
}

var (
  domain  = flag.String("domain", "evil.com", "the domain to harvest")
  dnsfile = flag.String("dnsfile", "/tmp/dat", "a file with hostnames")
  outfile = flag.String("outfile", "hosts.txt", "a file to store the discovered hostnames")
  //TODO: reverse lookup
  //TODO: netrange lookup
  debugOutput = flag.Bool("debug", false, "enable verbose output?")
)

func main() {

  runtime.GOMAXPROCS(runtime.NumCPU())

  // Command line options
  flag.Parse()
  setupLogging(log, debugOutput)

  if len(flag.Args()) == 3 {
    flag.Usage()
    os.Exit(1)
  }

  var wg sync.WaitGroup
  dispatcher := shared.NewDispatcher(5, wg)
  dispatcher.Run()

  // Reading DNS file
  dat, err := ioutil.ReadFile(*dnsfile)
  if err != nil {
    panic(err)
  }
  yourLinksSlice := strings.Split(string(dat), "\n")

  //check for wildcard
  wildcard := shared.CheckWildcard(shared.Task{Domain: *domain})

  // Processing all links by spreading them to `free` goroutines
  for _, host := range yourLinksSlice {
    task := &shared.Task{
      Domain:   *domain,
      Host:     host,
      Wildcard: wildcard,
    }

    log.Debugf("Sending Task for '%v'\n", task)
    //log.Printf("Main sending Task for '%s.%s'\n", task.Host, task.Domain)

    shared.TaskQueue <- task
  }

  close(shared.TaskQueue) //FIXME use a done channel
  log.Infof("Finished wordlist '%v'\n", shared.TaskQueue)

  //dispatcher.Stop()
  //pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

  //time.Sleep(1500 * time.Millisecond)
  //pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
  log.Debugf("Waiting for goroutines to finish '%v'\n", dispatcher.DispatcherWG)
  dispatcher.DispatcherWG.Wait()

  if false {
    reader := bufio.NewReader(os.Stdin)
    text, _ := reader.ReadString('\n')
    fmt.Println(text)

    pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
  }

}
