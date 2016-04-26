package shared

import (
  "fmt"
  "net"
  "strconv"
  "strings"

  "github.com/op/go-logging"
)

var log = logging.MustGetLogger("example")

type Task struct {
  Domain   string
  Host     string
  Wildcard *resultHost
}

func (t *Task) String() string {
  return fmt.Sprintf("%s.%s.", strings.ToLower(t.Host), strings.ToLower(t.Domain))
}

var TaskQueue = make(chan *Task, 1)
var ResultQueue = make(chan ResultRR, 1)

func (t *Task) lookupHosts() {
  r, err := net.LookupHost(t.String()) //TODO: problem with T-Online DNSerror

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Warningf("*** timeout error: %s\n", err.Error())
    } else if err.(*net.DNSError).Err == "no such host" {
      //log.Debugf("no such hostname: %s\n", err.Error() )
      return
    } else {
      log.Errorf("*** other error: %s\n", err.Error())
    }
  }

  log.Warningf("response '%v'\n", r)
}
func (t *Task) lookupIPs() {
  r, err := net.LookupIP(t.String())
  //r, err := net.LookupHost(t.String()) //TODO: problem with T-Online DNSerror

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Warningf("*** timeout error: %s\n", err.Error())
    } else if err.(*net.DNSError).Err == "no such host" {
      //log.Debugf("no such hostname: %s\n", err.Error() )
      return
    } else {
      log.Errorf("*** other error: %s\n", err.Error())
    }
  }
  //log.Warningf("Worker recvd Response '%v' for '%v'\n", r, t)

  for _, ansIp := range r {
    log.Debugf("wildcard?%v\t%v\n", t.Wildcard, ansIp)
    if t.Wildcard.IP.Equal(ansIp) {
      return
    }
  }

  for _, ansIp := range r {
    log.Debugf("received response '%v'\n", ansIp)

    //ansIp := net.ParseIP(ans)
    res := &resultHost{
      Domain: t.Domain,
      Host:   t.Host,
      IP:     ansIp,
      PTR:    "",
    }

    res.reverseLookup()
    //log.Infof("Worker reicvd result for '%s'\n", res)
    knownHostnames = append(knownHostnames, res.GetHostname())

    //push to results channel
    ResultQueue <- res
  }
}

func (t *Task) lookupCnames() {
  r, err := net.LookupCNAME(t.String())
  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Warningf("*** timeout error: %s\n", err.Error())
    } else if err.(*net.DNSError).Err == "no such host" {
      //log.Debugf("no such hostname: %s\n", err.Error() )
      return
    } else {
      log.Errorf("*** other error: %s\n", err.Error())
    }
  }
  log.Debugf("Worker recvd Response '%v' for '%v'\n", r, t)

  if strings.Compare(t.String(), r) != 0 {
    res := &resultCname{
      Domain: t.Domain,
      Host:   t.Host,
      Cname:  r,
    }

    //push to results channel
    ResultQueue <- res
  }
}

type ResultRR interface {
}

type resultHost struct {
  Domain string
  Host   string
  IP     net.IP
  PTR    string
}

func (r *resultHost) String() string {
  return fmt.Sprintf("%s.%s.\t%s\t%s", r.Host, r.Domain, r.IP, r.PTR)
}
func (r *resultHost) GetHostname() string {
  return fmt.Sprintf("%s.%s.", r.Host, r.Domain)
}

// A buffered channel that we can send work requests on.
//var TaskQueue chan Task

type resultCname struct {
  Domain string
  Host   string
  Cname  string
}

//https://stackoverflow.com/questions/15323767/does-golang-have-if-x-in-construct-similar-to-python
func stringInSlice(newHostname string, haystack []string) bool {
  for _, knownHostname := range haystack {
    log.Debugf("searchNeedle '%s':'%s'\n", newHostname, knownHostname)
    if knownHostname == newHostname {
      return true
    }
  }
  return false
}

var knownHostnames []string

func (r *resultHost) reverseLookup() {
  response, err := net.LookupAddr(r.IP.String())
  //r, err := net.LookupHost(rr.Ip.String())

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Debugf("*** timeout error: %s\n", err.Error())
    }
    // log.Printf("%v\n", err)
  }
  //TODO: only one response is present where 1+ exist: dig -x 193.30.192.26
  //log.Debugf("%v\t%v\n", r, response)

  //check wildcard
  for _, ans := range response {
    // log.Printf("%v\t%v\n", rr, ans)

    res := ans
    //add newly discovered hostnames
    //if strings.Contains(res, r.Domain) {
    log.Debugf("reverseLookup recvd result for '%s' and '%s'\n", res, r.Host)
    if !strings.HasPrefix(res, r.Host) {

      //only add if hostname is new
      if !stringInSlice(res, knownHostnames) {
        r.PTR = res
        log.Debugf("reverseLookup recvd result for '%s'\n", res)
      } else {
        log.Debugf("stringInSlice Domain not ok for '%s' and '%s'\n", res, knownHostnames)
      }
    }
    //}
  }
}

// Worker represents the worker that executes the task
type Worker struct {
  WorkerPool  chan chan *Task
  TaskChannel chan *Task
  quit        chan bool
}

func NewWorker(workerPool chan chan *Task) Worker {
  return Worker{
    WorkerPool:  workerPool,
    TaskChannel: make(chan *Task),
    quit:        make(chan bool)}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
  //defer w.Stop()

  go func() {
    for {

      // register the current worker into the worker queue.
      w.WorkerPool <- w.TaskChannel

      select {
      case task, more := <-w.TaskChannel:
        //log.Debugf("received %v\t%v", task, more)
        if more {
          // we have received a work request.
          log.Debugf("Worker '%p' recvd Task for '%s'\n", &w, task)

          task.lookupCnames()

          task.lookupIPs()

          //task.lookupHosts() //FIXME: Test me

        } else {
          //FIXME: Channel is empty before close
          log.Infof("End of TaskChannel, terminating worker '%v'", len(w.TaskChannel))
          // wait for quit signal
          <-w.quit
          return
        }
      case <-w.quit:
        // we have received a signal to stop
        log.Debugf("Worker '%p' stopped\n", &w)
        return
      }
    }
  }()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
  go func() {
    log.Debugf("Stopping Worker '%p'\n", &w)
    close(w.TaskChannel)
    w.quit <- true
  }()
}

func CheckWildcard(task Task) *resultHost {
  randInt := 123456890
  task.Host = strconv.Itoa(randInt)

  r, err := net.LookupHost(task.String()) //TODO: problem with T-Online DNSerror

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Infof("*** timeout error: %s\n", err.Error())
    }
    // log.Infof("%v\n", err)
  }

  //TODO: multiple ips
  for _, ans := range r {
    ansIp := net.ParseIP(ans)
    res := &resultHost{
      Domain: task.Domain,
      Host:   task.Host,
      IP:     ansIp,
      PTR:    "",
    }

    //TODO: lookup PTR record

    log.Infof("Wilcard detected: %v\n", res)
    return res
  }
  return &resultHost{
    Domain: "",
    Host:   "",
    IP:     net.ParseIP("0"),
    PTR:    "",
  }
}
