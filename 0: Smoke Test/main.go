package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

/*

	Write data to a server and read the same data back
	Accept TCP connections
	Data sent back unmodified
	Handle at least 5 simultaneous clients
	EOF marks end of data sent
	Close socket once data has been sent back
	Implement TCP echo service from RFC 862
	-> Echo server

	Improvements
	- Worker pool

	Logic
	- Worker pool to handle connections
	- Channel to share connections with workers
	- One worker per connection
	- Listener sends connection to channel
	- Jobs channel never closes
	- Worker closes connection


*/

func main() {

	const MAX_JOBS = 5
	const NUM_WKRS = 5

	// channel to deliver the connection to the go routine
	jobs := make(chan net.Conn, MAX_JOBS)

	// start workers, they wait until jobs are added
	for w := 1; w <= NUM_WKRS; w++ {
		fmt.Println("Started worker", w)
		go echoWorker(w, jobs)
	}

	// start tests, simulate sending to workers
	//for c := 1; c <= 30; c++ {
	//	time.Sleep(50 * time.Millisecond)
	//	go test()
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

func echoWorker(id int, jobs <-chan net.Conn) {
	// loop removes from the jobs channel
	// will run whenever there is something in the channel
	// runs/blocks until channel is closed
	for conn := range jobs {

		// socket closed once data sent

		// need a bigger buffer -> options include: smaller buf, append to large; ioutil.ReadAll
		// huge buffer for each read is not optimal...
		MAX_BUF := 4096
		buf := make([]byte, MAX_BUF)

		// data read loop, gets stuck if EOF not found
		for {
			var read int

			read, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("EOF reached, closing connection")
					break
				}
				panic(err)
			}
			writeToConn(conn, buf, read)
		}
		conn.Close()

	}
}

func writeToConn(c net.Conn, b []byte, read int) error {
	b = b[0:read] // likely faster than trim?
	fmt.Printf("Writing %v bytes to conn\n", read)
	_, err := c.Write(b)
	if err != nil {
		return err
	}
	return nil
}

// simulates sending data to the server
func test() {
	time.Sleep(2 * time.Second)
	conn, err := net.Dial("tcp", ":8123")
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	data := "testing"
	_, err = conn.Write([]byte(data))
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
