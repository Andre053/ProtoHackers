package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
)

/*
	Accept TCP connections
	Data from connection separated by new line
	Respond to connection until malformed response sent
	Messages are in JSON format
	Req malformed if not well-formed JSON, required field missing, method name not "isPrime" or number not number
	Res malformed if not well-formed JSON, required field missing, method name not "isPrime" or prime not bool

	Messages
	- Proper req: {"method": "isPrime", "number": <num>}
	- Proper res: {"method": "isPrime", "prime": "<bool>"}


	Need a worker pool
	Need a job channel


*/

type Request struct {
	Method *string `json:"method"`
	Number *int64  `json:"number"`
}

type RequestFloat struct {
	Method *string  `json:"method"`
	Number *float64 `json:"number"`
}

type Response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

type MalformedResponse struct {
	Method string `json:"malformed!"`
}

func main() {
	const MAX_JOBS = 8
	const NUM_WKRS = 8

	// channel to deliver the connection to the go routine
	jobs := make(chan net.Conn, MAX_JOBS)

	// start workers, they wait until jobs are added
	for w := 1; w <= NUM_WKRS; w++ {
		fmt.Println("Worker started", w)
		go worker(w, jobs)
	}

	// start tests, simulate sending to workers
	//for c := 1; c <= 5; c++ {
	//	time.Sleep(1000 * time.Millisecond)
	//	go test(c)
	//}

	PORT := ":8123"

	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		panic(err)
	}

	// listen loop
	for {
		conn, err := listener.Accept() // accept client connections
		fmt.Println("### Accepted a connection ###")
		fmt.Printf("Connection details:\n\tNetwork Address : %v\n", conn.RemoteAddr().String())
		if err != nil {
			panic(err)
		}
		jobs <- conn // send the connection to be worked

	}
}

// simulates sending data to the server
/*func test(num int64) {
	time.Sleep(2 * time.Second)
	conn, err := net.Dial("tcp", ":8123")
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	req := &Request{
		Method: "isPrime",
		Number: num,
	}
	data, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	_, err = conn.Write(data)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		if err == io.EOF {
			fmt.Println("EOF client")
		}
		panic(err)
	}
	//fmt.Println("Received", string(buf))
}*/

func checkPrime(p int64) (bool, error) {
	var i int64
	if p <= 1 {
		return false, nil
	}

	// might need to optimize
	for i = 2; i <= int64(math.Floor(float64(p)/2)); i++ {
		if p%i == 0 { // no remainder
			return false, nil
		}
	}
	return true, nil

}

func worker(id int, jobs <-chan net.Conn) {
	// loop removes from the jobs channel
	// will run whenever there is something in the channel
	// runs/blocks until channel is closed

	sep := []byte("\n") // ASCII 10

	for conn := range jobs {
		fmt.Printf(">Session started on worker %v<\n", id)
		MAX_BUF := 200000
		buf := make([]byte, MAX_BUF)

		for {
			read, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Disconnect client due to EOF reached")
					break
				}
				panic(err)
			}
			if read == MAX_BUF {
				// there could be a started entry that is finished in next read
				// find a way to parse out the entry, or make huge buffer
				fmt.Println("Filled buffer")
			}

			bufTrimmed := bytes.Trim(buf, "\x00")

			reqsTooLong := bytes.Split(bufTrimmed, sep)
			reqs := reqsTooLong[:len(reqsTooLong)-1] // remove last entry

			// first check for malformed
			r := reqs[0]
			fmt.Printf("Request: %v", string(r))
			req := &Request{}
			err = json.Unmarshal(r, req)
			if err != nil { // must be malformed, kill it

				reqFloat := &RequestFloat{}
				err = json.Unmarshal(r, reqFloat)
				if err == nil {
					fmt.Printf(" is float\n")
					res := createRes(false)
					err = sendJson(conn, res)
					if err != nil {
						panic(err)
					}
					break

				}
				fmt.Printf(" is malformed\n")
				res := createMalformed()
				err = sendJson(conn, res)
				if err != nil {
					panic(err)
				}

				break

			}
			if req.Method == nil || req.Number == nil || *req.Method != "isPrime" {
				fmt.Printf(" is malformed\n")
				res := createMalformed()
				err = sendJson(conn, res)
				if err != nil {
					panic(err)
				}
				break
			}
			num := req.Number
			isPrime, err := checkPrime(*num)
			if err != nil {
				// send malformed request
				fmt.Printf(" is malformed\n")
			}
			//fmt.Printf("Valid isPrime result: %v\n", isPrime)
			// send response with result
			res := createRes(isPrime)
			err = sendJson(conn, res)
			if err != nil {
				panic(err)
			}
			fmt.Printf(" is valid\n")
			// check remaining entries
			for _, r := range reqs[1:] {
				fmt.Println("request:", string(r))
				req := &Request{}
				err = json.Unmarshal(r, req)
				if err != nil { // must be a syntax error, continue

					continue
				}
				// no error parsing, so request is valid
				num := req.Number
				isPrime, err := checkPrime(*num)
				if err != nil {
					// send malformed request
				}
				//fmt.Printf("Valid isPrime result: %v\n", isPrime)
				// send response with result
				res := createRes(isPrime)
				err = sendJson(conn, res)
				if err != nil {
					panic(err)
				}
				//fmt.Printf("Request %v handled\n\n", string(r))

			}
			//fmt.Printf("Completed all requests in buffer\n\n")
		}
		fmt.Println(">Closing connection<")
		conn.Close()

	}
}

func sendJson(c net.Conn, r any) error {
	jsonRes, err := json.Marshal(r)
	if err != nil {
		return err
	}
	//fmt.Println("Sending JSON:", string(jsonRes))
	jsonRes = append(jsonRes, byte('\n')) // each res must be terminated by a newline
	err = sendToConn(c, jsonRes)
	if err != nil {
		return err
	}
	return nil
}

func createMalformed() Response {
	res := Response{
		Method: "error",
	}
	return res
}

func createRes(isPrime bool) Response {
	res := Response{
		Method: "isPrime",
		Prime:  isPrime,
	}
	return res
}

func sendToConn(c net.Conn, data []byte) error {
	_, err := c.Write(data)
	if err != nil {
		return err
	}
	return nil
}
