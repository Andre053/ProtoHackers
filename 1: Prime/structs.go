package main

type Request struct {
	Method     string  `json:"method"`
	NumberByte float64 `json:"number"`
}

type Response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

type MalformedResponse struct {
	Method string `json:"malformed!"`
}
