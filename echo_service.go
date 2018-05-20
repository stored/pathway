package main

import (
	"context"
	"encoding/json"
	"log"
)

func (s *EchoService) Message(ctx context.Context, payload json.RawMessage) (*json.RawMessage, *Response, error) {
	log.Println("Echo.Message called")
	data := new(json.RawMessage)
	json.Unmarshal(payload, &data)
	return data, nil, nil
}
