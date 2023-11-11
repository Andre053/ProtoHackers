package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
)

func main() {
	run()
}

func run() {
	const MAX_JOBS = 8
	const NUM_WKRS = 8

	jobs := make(chan net.Conn, MAX_JOBS)

	for w := 1; w <= NUM_WKRS; w++ {
		go worker(w, jobs)
	}

	PORT := ":8123"

	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		panic(err)
	}
	fmt.Println("Starting to listen...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("### Accepted a connection")

		jobs <- conn // send connection to be worked
	}
}

func worker(id int, jobs <-chan net.Conn) {
	sep := []byte("\n")

	for conn := range jobs {
		defer conn.Close()
		handleConn(conn, sep, id)
		fmt.Printf("Connection handled by %v\n", id)
	}
}

func handleConn(conn net.Conn, sep []byte, id int) {
	count := 1

	scanner := bufio.NewScanner(conn)

	// loop to read the connection data
	for {

		// only need a single scan
		scanner.Scan()
		reqStr := scanner.Text()

		err := scanner.Err()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
		}

		req := []byte(reqStr)
		// have data, check if valid request
		request, valid := isValid(req)
		if !valid {
			// handle invalid request
			fmt.Printf("\nScanned:\t%v\nData received:\t%v\nMalformed...\nWorker %v count:\t%v\n", reqStr, string(req), id, count)
			sendMalformed(conn)
			count += 1
			break // once malformed is seen, stop processing conn
		}

		isPrime := checkNumber(*request)

		res := createRes(isPrime)
		fmt.Printf("\nScanned:\t%v\nData received:\t%v\n%v prime?\t%v\nWorker %v count:\t%v\n", reqStr, string(req), big.NewFloat(request.NumberByte).String(), isPrime, id, count)
		err = sendJson(conn, res)
		if err != nil {
			panic(err)
		}
		count += 1

	}
	fmt.Printf("Total requests for worker %v:\t%v\n", id, count-1)
}

// much too slow it seems
// should be able to handle any size number, but float64 and uint64 seem to be sufficient for this exercise
func checkPrimeSlow(num *big.Int) bool {

	limit := new(big.Int)
	idx := big.NewInt(1)
	remainder := new(big.Int)

	if num.Cmp(big.NewInt(2)) == 0 || num.Cmp(big.NewInt(3)) == 0 {
		//fmt.Printf("Yes, by base case 2 or 3\n")
		return true
	}

	if num.Cmp(big.NewInt(1)) != 1 {
		//fmt.Printf("No, by base case <=1\n")

		return false
	}

	limit.Div(num, big.NewInt(2))
	limit.Add(limit, big.NewInt(1))

	for idx = big.NewInt(2); idx.Cmp(limit) != 0; idx.Add(idx, big.NewInt(1)) {
		remainder.Mod(num, idx)
		if remainder.Cmp(big.NewInt(0)) == 0 {
			//fmt.Printf("No, %v is a factor\n", idx)
			return false
		}
	}

	//fmt.Printf("Yes, by brute force\n")
	return true
}

func checkNumber(req Request) bool {
	//numString := string(req.NumberByte)
	//numFloat := new(big.Float).SetPrec(uint(len(numString)))
	//numFloat.SetString(numString)

	// smaller num route
	// have a float64
	if req.NumberByte != float64(int(req.NumberByte)) && req.NumberByte < 0 {
		return false
	}
	numUint64 := uint64(req.NumberByte)
	return checkPrimeFast(numUint64)
	/*

		// big num route
		numFloat := big.NewFloat(req.NumberByte)

		num := new(big.Int)
		num, accuracy := numFloat.Int(num)
		if accuracy != 0 {
			// not an integer
			return false
		}

		return checkPrimeSlow(num)
	*/

}

func checkPrimeFast(num uint64) bool {
	if num == 0 || num == 1 {
		return false
	}
	if num == 2 {
		return true
	}

	for i := uint64(2); i < num/2+1; i++ {
		if num%i == 0 {
			return false
		}

	}
	return true
}

func isValid(reqData []byte) (*Request, bool) {

	req := &Request{}
	err := json.Unmarshal(reqData, req)
	if err != nil { // does not fit
		return nil, false
	}

	// there may be faster checks
	if req.Method != "isPrime" || !bytes.Contains(reqData, []byte("number")) {
		return nil, false
	}
	return req, true
}

func extractRequests(data []byte, sep []byte) [][]byte {
	// trim all zeros
	bufTrimmed := bytes.Trim(data, "\x00")

	// split data into a list of requests
	reqs := bytes.Split(bufTrimmed, sep)

	return reqs[:len(reqs)-1]
}
