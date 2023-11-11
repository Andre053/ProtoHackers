package main

import (
	"encoding/json"
	"net"
)

func sendMalformed(conn net.Conn) {
	mf := createMalformed()
	err := sendJson(conn, mf)
	if err != nil {
		panic(err)
	}
}

func sendJson(c net.Conn, r any) error {
	jsonRes, err := json.Marshal(r)
	if err != nil {
		return err
	}
	jsonRes = append(jsonRes, byte('\n')) // each res must be terminated by a newline
	//fmt.Println("Sending:\t", string(jsonRes))
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
