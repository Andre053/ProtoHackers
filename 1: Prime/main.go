package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
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
	Method string `json:"method"`
	Number int    `json:"number"`
}

type Response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

type MalformedResponse struct {
	Method string `json:"malformed!"`
}

func main() {
	const MAX_JOBS = 5
	const NUM_WKRS = 5

	// channel to deliver the connection to the go routine
	jobs := make(chan net.Conn, MAX_JOBS)

	// start workers, they wait until jobs are added
	for w := 1; w <= NUM_WKRS; w++ {
		go worker(w, jobs)
	}

	// start tests, simulate sending to workers
	for c := 1; c <= 5; c++ {
		time.Sleep(1000 * time.Millisecond)
		go test(c)
	}

	PORT := ":8123"

	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		panic(err)
	}

	// listen loop
	for {
		conn, err := listener.Accept() // accept client connections
		if err != nil {
			panic(err)
		}
		jobs <- conn // send the connection to be worked
	}

}

// simulates sending data to the server
func test(num int) {
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

}

func checkPrime(p int) (bool, error) {
	if p == 1 {
		return true, nil
	}
	if p < 1 {
		return false, errors.New("Number is invalid")
	}

	// might need to optimize
	for i := 2; i < p/2; i++ {
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
	for conn := range jobs {

		buf := make([]byte, 1024)

		_, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF:", err)
				break
			}
			panic(err)
		}

		// split each newline
		sep := []byte("\n") // ASCII 10
		reqs := bytes.Split(buf, sep)

		for _, r := range reqs {
			// need to trim r
			r = bytes.Trim(r, "\x00")
			fmt.Println("Received request:", string(r))
			req := &Request{}
			err = json.Unmarshal(r, req)
			if err != nil { // must be a syntax error, send malformed
				fmt.Println("Error parsing", string(r), "as request:", err)
				res := createMalformed()
				sendJson(conn, res)
				if err != nil {
					panic(err)
				}
				continue
			}
			// no error parsing, so request is valid
			num := req.Number
			isPrime, err := checkPrime(num)
			if err != nil {
				// send malformed request
				fmt.Println("Error in checkPrime:", err)
			}
			// send response with result
			res := createRes(isPrime)
			err = sendJson(conn, res)
			if err != nil {
				panic(err)
			}
		}
		conn.Close()
	}
}
func sendJson(c net.Conn, r any) error {
	jsonRes, err := json.Marshal(r)
	if err != nil {
		return err
	}
	fmt.Println("Sending JSON:", string(jsonRes))
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
