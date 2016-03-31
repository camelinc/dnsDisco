package main

//TODO: AXFR
//TODO: netrange

import (
  "net"
  "flag"
  "log"
  "runtime"
  "time"
  "sync"
  "strings"
  "fmt"
  "os"
  "bufio"
  "io/ioutil"
)
const (
	TIMEOUT time.Duration = 3 // seconds
)
// pipe log to devnull
type DevNull struct{}
func (DevNull) Write(p []byte) (int, error) {
	return len(p), nil
}

type DNSentry struct {
  Hostname string
  Ip net.IP
  PTR string
}
//https://stackoverflow.com/questions/15323767/does-golang-have-if-x-in-construct-similar-to-python
func stringInSlice(newHostname string, haystack []string) bool {
	for _, knownHostname := range haystack {
		if knownHostname == newHostname {
			return true
		}
	}
	return false
}

func checkWildcard(domain string) DNSentry  {
  randInt := 1234567
  rec := fmt.Sprintf("%d.%s", randInt, domain)

  r, err := net.LookupHost(rec) //TODO: problem with T-Online DNSerror

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Printf("*** timeout error: %s\n", err.Error())
    }else if outputVerbose {
      log.Printf("*** error: %s\n", err.Error())
    }
    // log.Printf("%v\n", err)
  }

  for _, ans := range r{
    ansIp := net.ParseIP(ans)
    res := DNSentry{rec, ansIp, ""}
    log.Printf("Result: %v\n", res)
    return res
  }
  return DNSentry{"", net.ParseIP("0"), ""}
}

func lookupIpAddress(rr DNSentry, domain string, linkChan chan string) string {
  r, err := net.LookupAddr(rr.Ip.String())
  //r, err := net.LookupHost(rr.Ip.String())

  if err != nil {
    if err.(*net.DNSError).IsTimeout {
      log.Printf("*** timeout error: %s\n", err.Error())
    }
    // log.Printf("%v\n", err)
  }
  //TODO: only one response is present where 1+ exist: dig -x 193.30.192.26
  // log.Printf("%v\t%v\n", rr, r)

  var res string
  //check wildcard
  for _, ans := range r{
    // log.Printf("%v\t%v\n", rr, ans)

    res = ans
    //add newly discovered hostnames
    if strings.Contains(res, domain) {
      if !strings.HasPrefix(res, rr.Hostname) {
        //only add if hostname is new
        if !stringInSlice(res, knownHostnames) {
          linkChan <- res
        }
      }
    }
  }
  return res
}
func worker(domain string, linkChan chan string, done chan string) <-chan DNSentry{
  // fmt.Printf("Starting goroutine\n")
  rCh := make(chan DNSentry, 5)

  // prepare workers
  go func() {
    var wg sync.WaitGroup

    if outputVerbose {
      log.Printf("Starting %d goroutines\n",runtime.NumCPU())
    }

    // perform wildcard check
    //checkWildcard(domain)
    //return
    wildcard := checkWildcard(domain)

    for i := 0; i < runtime.NumCPU(); i++ {
      wg.Add(1)

      go func() {
        if outputVerbose {
          log.Printf("Started goroutine\n")
        }

SendRequestLoop:
        //for {
        //  j, url := <-linkChan {
        for url := range linkChan {
          if outputVerbose {
            log.Printf("Started processing %s\n", url)
          }

          rec := url
          r, err := net.LookupHost(rec) //TODO: problem with T-Online DNSerror

          if err != nil {
            if err.(*net.DNSError).IsTimeout {
              log.Printf("*** timeout error: %s\n", err.Error())
            }else if outputVerbose {
                log.Printf("*** error: %s\n", err.Error())
            }
            // log.Printf("%v\n", err)
          }

          //check wildcard
          for _, ans := range r{
            ansIp := net.ParseIP(ans)
            res := DNSentry{rec, ansIp, ""}
            //res := DNSentry{rec, ans}
            // log.Printf("wildcard?%v\t%v\n", wildcard, res)

            if wildcard.Ip.Equal(ansIp) {
              if outputVerbose {
                log.Printf("wildcard!!!%v\t%v\n", wildcard, res)
              }
              continue SendRequestLoop
            }
          }


          // Stuff must be in the answer section
          for _, ans := range r{
            ansIp := net.ParseIP(ans)
            res := DNSentry{rec, ansIp, ""}
            res.PTR = lookupIpAddress(res, domain, linkChan)
            // log.Printf("Result: %v\n", res)
            knownHostnames = append(knownHostnames, rec)
            rCh <- res
          }

          //TODO: CNAME
        }
        if outputVerbose {
          log.Printf("Done processing goroutine: %v\n",wg)
        }
        wg.Done()
      }()

    }
    // Waiting for all goroutines to finish (otherwise they die as main routine dies)
    //  and close the result channel
    go func() {
      wg.Wait()
      close(rCh)
    }()
    // fmt.Printf("Waiting To Finish: %v\n",wg)
    wg.Wait()
  }()
  return rCh
}

var outputVerbose bool
var knownHostnames []string
var networkRanges []string

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  //log.SetOutput(new(DevNull))
  // net.dnsReadConfig
  // fmt.Printf("%v\n",net)

  var wg sync.WaitGroup
  lCh := make(chan string, 5)

  // worker closes the done channel when it returns; it may do so before
  // receiving all the values from c and errc.
  done := make(chan string)
  defer close(done)

  // Command line options
  var domain string
  flag.StringVar(&domain, "domain", "evil.com", "the domain to harvest")
  var dnsfile string
  flag.StringVar(&dnsfile, "dnsfile", "/tmp/dat", "a file with hostnames")
  var outfile string
  flag.StringVar(&outfile, "outfile", "hosts.txt", "a file to store the discovered hostnames")
  //TODO: reverse lookup
    //TODO: netrange lookup
  flag.BoolVar(&outputVerbose, "verbose", false, "enable verbose output?")
  flag.Parse()

  if len(flag.Args()) == 3 {
    flag.Usage()
    os.Exit(1)
  }

  // Reading DNS file
  dat, err := ioutil.ReadFile(dnsfile)
  if err != nil {
    panic(err)
  }
  yourLinksSlice := strings.Split(string(dat), "\n")

  // prepare working list
  lCh <- domain
  var hostname string
  go func() {
    // Processing all links by spreading them to `free` goroutines
    for _, link := range yourLinksSlice {
      hostname = fmt.Sprintf("%s.%s.", link, domain)
      if outputVerbose {
        fmt.Printf("Sending %s\n",hostname)
      }
      lCh <- hostname
    }
  	close(lCh)
    // log.Printf("Finished wordlist\n")
  }()

  // worker2
  rCh := worker(domain, lCh, done)

  //TODO: results gatherer in separate goroutine
  // open outfile
  f, err := os.Create(outfile)
  w := bufio.NewWriter(f)

  // process results
  for res := range rCh {
    outString := fmt.Sprintf("%-15s\t%s\t%s\n", res.Ip, res.Hostname, res.PTR)
    fmt.Printf(outString)

    //verify routable
    // https://golang.org/pkg/net/#IP.IsGlobalUnicast
    if res.Ip.IsGlobalUnicast() {
      // get network range for res.Ip
      cidr := fmt.Sprintf("%s/24",res.Ip)
      _, networkRange, _ := net.ParseCIDR(cidr)
      //lookup at RIPE:
      //  https://github.com/RIPE-NCC/whois/wiki/WHOIS-REST-API
      //  https://rest.db.ripe.net/search?&query-string=78.47.227.0
      //  https://rest.db.ripe.net/search?type-filter=inetnum&query-string=netto
      //log.Printf("network range: %v\n", networkRange)
      networkRangeS := networkRange.String()
      if !stringInSlice(networkRangeS, networkRanges) {
        networkRanges = append(networkRanges, networkRangeS)
      }
      //log.Printf("network ranges: %v\n", networkRanges)

      _, err := w.WriteString(outString)
      if err != nil {
        panic(err)
      }
      w.Flush()

    }else{
      log.Printf("Not a global Unicast: %v\n", res.Ip)
    }
  }

  // Closing channel (waiting in goroutines won't continue any more)

  // Waiting for all goroutines to finish (otherwise they die as main routine dies)
  wg.Wait()
  // log.Printf("Waiting To Finish: %v\n",wg)
  wg.Wait()

  fmt.Printf("\nPrinting likely network ranges:\n")
  for _, networkRange := range networkRanges {
    outString := fmt.Sprintf("%s\n", networkRange)
    fmt.Printf(outString)
  }

  //log.Println("\nTerminating Program")
}
