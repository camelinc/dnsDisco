
Wonderful DNS Brute force guessing tool implemented in golang.

# ToDos

* [x] implement dispatcher pattern
* [ ] Perform action based on result type
  * [ ] Properly report CNAMES
    * [ ] only last in chain is returned by net.LookupCNAME
  * [ ] Parse TXT record
  * [ ] add newly discovered domains to Scope after user action
* [ ] Flag to get http response header
  * [ ] ssdeep on http response body/header
    * see [gossdeep](https://github.com/dutchcoders/gossdeep)
* [ ] Gather netrange information
  * [ ] bgp.he.net.
  * [ ] [WHOIS REST API Search](https://github.com/RIPE-NCC/whois/wiki/WHOIS-REST-API)
    * `curl "http://rest.db.ripe.net/search?source=ripe&query-string=127.0.0.1&flags=no-filtering&flags=no-referenced"`
* [ ] print to results to file
  * [ ] CSV
  * [ ] XML
* [ ] utilise 3rd party services to get moar information
  * [ ] [censys](https://github.com/jgamblin/censys)
  * [ ] [scans.io](https://scans.io)
    * [ ] [json index](https://scans.io/json)
    * [ ] [Reverse DNS](https://scans.io/study/sonar.rdns)
    * [ ] [DNS Records (ANY)](https://scans.io/study/sonar.fdns)
    * [ ] [Scan for AXFR DNS replies](https://scans.io/study/hanno-axfr)
    * [ ] [Zonefile Database](https://scans.io/study/axfr-research)


# Links
## Related worker
* [gobuster](https://github.com/OJ/gobuster)
* [subbrute](https://github.com/TheRook/subbrute)
* [fierce](http://tools.kali.org/information-gathering/fierce)
* [dnsrecon](https://github.com/darkoperator/dnsrecon)
* [dnstwist](https://github.comelceef/dnstwist)

## Golang
* [Writing worker queues, in Go](https://nesv.github.io/golang/2014/02/25/worker-queues-in-go.html)
* [Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/)
* https://golang.org/pkg/net/
* https://stackoverflow.com/questions/34729158/how-to-detect-if-two-golang-net-ipnet-objects-intersect
