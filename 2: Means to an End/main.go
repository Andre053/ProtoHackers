package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
)

/*
TCP server for clients to insert and query timestamped prices
Each client tracks price of diff asset
Clients send messages to server to insert or query prices
Conn from client is separate session
Sessions data rep diff asset, each session can only query data supplied by itself

OVERVIEW
- Client tracks price of a *different* asset
- Clients send messages to *insert or query* the prices
- Each connection is in a separate session
- Each session's data represents a different asset
- Each session can only query the data supplied by itself (separate channels)


MESSAGE FORMAT
- Binary format
- Message from a client is 9 bytes long
	- 1st byte: Indicates type: ASCII uppercase 'I' or 'Q', insert or query
	- 2-8 bytes: Two 32-bit signed 2's complement integers (big endian order)
- Multiple messages per connection
- Messages *not* delimited

INSERT
- Insert a timestamped price
- 1st int32 is a timestamp (unix time)
- 2nd int32 is the price (time is price)
	- May occur out-of-order
	- Prices can go negative
	- Behavior is undefined if multiple prices with the same timestamp from the same client

QUERY
- Query the *average price over a given period of time*
- Format is two timestamps, mintime and maxtime
- Server must compute the mean of inserted prices between timestamps
	- If mean is not an integer, round up or down at server's discretion
- Server must send the mean to the client as a single int32

RQUIREMENTS
- Handle 5 simultaneous clients
- When undefined, do anything for that client (sendj


PLANNING
- Start 5 workers
- Each connection holds a map with the values, ideally ordered for binary search
	- Hold two data structures:
		-> Map with timestamp to value for O(1) lookups
		-> Ordered list of timestamps for O(nlogn)
*/

type messageInsert struct {
	Timestamp int32
	Price     int32
}
type messageQuery struct {
	TimestampStart int32
	TimestampEnd   int32
}

type assetTracker struct {
	timestamps timestamps      // list of timestamps for the asset
	data       map[int32]int32 // timestamps to the data
}
type timestamps []int32

// implementing sort interface for timestamps
func (t timestamps) Len() int {
	return len(t)
}
func (t timestamps) Less(i, j int) bool {
	return t[i] < t[j]
}
func (t timestamps) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// this is hella slow but should work
func (at assetTracker) AddTimestamp(t int32, p int32) {
	// add to timestamps and order it for lookups
	at.timestamps = append(at.timestamps, t)
	sort.Sort(at.timestamps) // sorts as reference

	// add to map for quick value lookups
	_, exists := at.data[t]
	if exists {
		fmt.Println("Complete undefined behavior!")
	} else {
		at.data[t] = p
	}
}

func (at assetTracker) BinSearch(v int32) int {
	low, high := 0, at.timestamps.Len()

	for low <= high {
		mid := low + (high-low)/2

		if at.timestamps[mid] == v {
			return mid
		}
		if at.timestamps[mid] < v {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return -1 // not found
}

func (at assetTracker) ComputeAverage(t1, t2 int32) int32 {
	// grab range of timestamps of interest
	var start, end int
	var sum int32
	sum = 0

	start = at.BinSearch(t1)
	end = at.BinSearch(t2)

	for i := start; i < end; i++ {
		sum += at.data[at.timestamps[i]]
	}
	return sum / int32(end-start)
}

func main() {

	fmt.Println("Means to an End")
}

func run(WORKERS int) {

	// channel to deliver the connection to the go routine
	jobs := make(chan net.Conn, WORKERS)
	for i := 0; i < WORKERS; i++ {
		go worker(jobs)
	}

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

func worker(jobs chan net.Conn) {
	for conn := range jobs {
		MAX_BUF := 9
		buf := make([]byte, MAX_BUF)

		// data read loop
		for {
			var read int

			read, err := conn.Read(buf)
			fmt.Printf("Read %v bytes\n", read)
			if err != nil {
				if err == io.EOF {
					fmt.Println("EOF reached, closing connection")
					break
				}
				panic(err)
			}
			err = handleRequest(buf)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func handleRequest(buffer []byte) error {
	if buffer[0] == byte('Q') { // query
		return handleQuery(buffer[1:])
	} else if buffer[0] == byte('I') { // insert

		return handleInsert(buffer[1:])
	}
	return nil
}

func handleQuery(buffer []byte) error {
	query := &messageQuery{
		TimestampStart: int32(binary.BigEndian.Uint32(buffer[:4])),
		TimestampEnd:   int32(binary.BigEndian.Uint32(buffer[5:])),
	}
	fmt.Printf("Query received, start: %v; end: %v\n", query.TimestampStart, query.TimestampEnd)
	return nil
}

func handleInsert(buffer []byte) error {
	insert := &messageInsert{
		Timestamp: int32(binary.BigEndian.Uint32(buffer[:4])),
		Price:     int32(binary.BigEndian.Uint32(buffer[5:])),
	}
	fmt.Printf("Insert received, timestamp: %v; price: %v\n", insert.Timestamp, insert.Price)
	return nil
}
