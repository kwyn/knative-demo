package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

    "github.com/bojand/ghz/runner"
)

func main() {
	var output = flag.String("o", fmt.Sprintf("load-test-report-%v.csv", time.Now()), "report output file")
	var ip = flag.String("ip", "", "ip of ingress")
	var host = flag.String("host", "localhost", "host to authorize the request with")
	flag.Parse()
	fmt.Printf("IP %v and host %v\n", *ip, *host)

	file, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
    defer w.Flush()

	var n uint = 1
	for n < 400 {
		fmt.Printf("Testing %v concurrent requests\n", n)
		rps := run(n, time.Minute*2, *ip, *host)
		fmt.Printf("%v RPS for %v concurrency\n", rps, n)
		if n < 50 {
			n = n + n
		} else {
			n = n + n/2
		}
		err := w.Write([]string{strconv.FormatUint(uint64(n), 10), strconv.FormatFloat(rps, 'f', 5, 64)})
		if err != nil {
			panic(err)
		}
		// Let service cool off
		time.Sleep(time.Minute * 1)
	}

	fmt.Println("Done!")
}

// Run runs the load tester with n conncurrent connections as fast as possible for d duration
func run(n uint, d time.Duration, ip string, host string) float64 {
	start := time.Now()
	//  TODO (kwyn): eventually allow certs to be a part of this request and to allow insecure.
	// Trying to recreate the following command
	// ` ghz
	//      -c 3
	//      --connections 3
	//      -z 5s
	//      --proto ./ping.proto
	//      --insecure
	//      --call ping.PingService.Ping
	//      --authority grpc-ping.default.example.com
	//      35.247.54.84:80`
	report, err := runner.Run(
		"ping.PingService.Ping",
		fmt.Sprintf("%v:80", ip),
		runner.WithProtoFile("./proto/ping.proto", []string{}),
		runner.WithTotalRequests(math.MaxInt32),
		runner.WithAuthority(host),
		runner.WithInsecure(true),
		runner.WithConnections(n),
		runner.WithConcurrency(n),
		runner.WithRunDuration(d),
		runner.WithDataFromJSON("{\"msg\":\"hello\" }"),
	)

	fmt.Println("End Reason: ", report.EndReason)
	fmt.Println("End Reason: ", report.Total)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	end := time.Now()
	fmt.Println("actual duration", end.Sub(start))
	return report.Rps
}
