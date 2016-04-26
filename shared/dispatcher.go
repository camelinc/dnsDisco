package shared

import (
  "fmt"
  "sync"
  "time"
)

var (
  //MaxWorker = os.Getenv("MAX_WORKERS")
  MaxWorker = 5
  //MaxQueue  = os.Getenv("MAX_QUEUE")
  MaxQueue = 5
)

// https://gist.github.com/nesv/9233300
type Dispatcher struct {
  // A pool of workers channels that are registered with the dispatcher
  WorkerPool   chan chan *Task
  WorkerSlice  []Worker
  WorkerWG     *sync.WaitGroup
  DispatcherWG *sync.WaitGroup
  quit         chan bool
}

func NewDispatcher(maxWorkers int, myWG sync.WaitGroup) *Dispatcher {

  return &Dispatcher{
    WorkerPool:   make(chan chan *Task, maxWorkers),
    WorkerWG:     new(sync.WaitGroup),
    DispatcherWG: new(sync.WaitGroup),
    quit:         make(chan bool),
  }
}

func (d *Dispatcher) Run() {
  // starting n number of workers
  for i := 0; i < MaxWorker; i++ {
    worker := NewWorker(d.WorkerPool)

    log.Debugf("Dispatcher started new Worker '%v'\n", worker)
    worker.Start()
    d.WorkerSlice = append(d.WorkerSlice, worker)
  }

  go d.dispatch()
  go d.gather()
}

func (d *Dispatcher) dispatch() {
  log.Debugf("Dispatcher started dispatching\n")

  for {
    select {
    case task, more := <-TaskQueue:
      //log.Debugf("received '%v'",task)
      if more {
        // a task request has been received
        go func(task *Task) {
          d.WorkerWG.Add(1)
          defer d.WorkerWG.Done()
          // try to obtain a worker task channel that is available.
          // this will block until a worker is idle
          taskChannel := <-d.WorkerPool

          // dispatch the task to the worker task channel
          taskChannel <- task
        }(task)
      } else {
        log.Infof("received all jobs '%v'", TaskQueue)
        defer d.Stop()
        return
      }
    }
  }
}

func (d *Dispatcher) gather() {
  d.DispatcherWG.Add(1)
  defer d.DispatcherWG.Done()
  log.Debugf("Waiting for goroutines to finish '%v'\n", d.DispatcherWG)

  for {
    select {
    case res, more := <-ResultQueue:
      log.Debugf("received %v\t%v", res, more)

      if more {
        switch res := res.(type) {
        case *resultHost:
          go func(r *resultHost) {
            //FIXME: write to file
            fmt.Printf("%s.%s\t%s\t%s\n", r.Host, r.Domain, r.IP, r.PTR)
          }(res)

        case *resultCname:
          go func(r *resultCname) {
            //FIXME: write to file
            fmt.Printf("%s.%s\t%s\n", r.Host, r.Domain, r.Cname)
          }(res)

        default:
          log.Warningf("Dont know what we got here >%v<\n", res)

        }
      } else if res == nil {
        log.Debugf("Dispatcher.Result waiting for quit '%p'\n", &d)
        x := <-d.quit
        // we have received a signal to stop
        log.Debugf("Dispatcher.Result '%v' stopped\n", x)
        return
      } else {
        time.Sleep(100 * time.Millisecond)
        continue
      }

    case <-d.quit:
      // we have received a signal to stop
      log.Debugf("Dispatcher.Result '%p' stopped\n", &d)
      return
    }
  }
}

// Stop signals the worker to stop listening for work requests.
func (d *Dispatcher) Stop() {
  log.Debugf("Waiting for goroutines to finish '%v'\n", d.WorkerWG)
  d.WorkerWG.Wait()

  close(ResultQueue)
  d.quit <- true

  // starting n number of workers
  for _, worker := range d.WorkerSlice {
    log.Debugf("Stopping worker '%v'\n", worker)
    go worker.Stop()
  }
  //close(d.WorkerPool)
  log.Debugf("Stopped dispatcher '%v'\n", d)
}
