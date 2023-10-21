package main

import (
	"fmt"
)

/*
TCP server for clients to insert and query timestamped prices
Each client tracks price of diff asset
Clients send messages to server to insert or query prices
Conn from client is separate session
Sessions data rep diff asset, each session can only query data supplied by itself

*/

func main() {

	fmt.Println("Means to an End")
}
