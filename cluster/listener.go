package cluster

import (
	"fmt"
	"net"
	"encoding/json"
)

type ListenerInterface struct {
	address string
        ln	*net.TCPListener
}

func newListenerInterface(addr string) (*ListenerInterface, error) {
	fmt.Println("ListenerInterface!!!", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("----")
	}
	return &ListenerInterface{addr, ln.(*net.TCPListener)}, nil
}

func (s *ListenerInterface) Accept() {//(c net.Conn, err error){
	for{
		connIn, error := s.ln.Accept()
		if error != nil {
			if _, ok := error.(net.Error); ok {
				fmt.Println("Error received while listening.")
			}
		} else {
			var requestMessage string
			json.NewDecoder(connIn).Decode(&requestMessage)
			responseMessage := "Sure buddy.. too easy.. from " + s.address
			json.NewEncoder(connIn).Encode(&responseMessage)
			fmt.Println(requestMessage)
		}
	}
}

// Close closes the service.
func (s *ListenerInterface) Close() {
	s.ln.Close()
	return
}

