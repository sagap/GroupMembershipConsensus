package main

import (
	"flag"
	"github.com/andelf/go-curl"
	"fmt"
	"net/url"
	"log"
	"net"
	"encoding/json"
	"strings"
)
type clientHandler struct{
	client *clientInfo
}

type clientInfo struct{
	myIp       string
	Id	   string
}

type ListenerInterface struct {
	address string
	ln	*net.TCPListener
}


var sent = false

const REQUEST_GROUPS string = "which request groups"
const REQUEST_MEMBERS string= "which members"

func main(){
	clientId := flag.String("id", "1", "ID")
	addressIP := flag.String("address", "http://127.0.0.1:10001", "address")
	toSend := flag.String("clusterAddress", "http://127.0.0.1:22380", "ClusterAddress")
	flag.Parse()
	fmt.Println("Client info: ",*clientId," ",*addressIP)
	var client1 clientInfo
	client := client1.NewClient(*clientId,*addressIP)
	url, err := url.Parse(*addressIP)
	if err != nil {
		log.Fatalf("raftexample: Failed parsing URL (%v)", err)
	}
	ln, err := client.newListenerInterface(url.Host)
	easy := curl.EasyInit()
	Request(easy,"group","",*toSend)
	var leave, choice1, groupName, register string
	//defer easy.Cleanup()
	fmt.Println("Do you want to list members of a group?")
	fmt.Scanf("%s",&choice1)
	if strings.Compare(choice1,"Y") ==0{
		fmt.Println("Members from which group?")
		fmt.Scanf("%s",&groupName)
		Request(easy,"members",groupName,*toSend)
	}
	fmt.Println("Choose group to register")
	fmt.Scanf("%s",&register)
	client.Register(easy,register,*toSend)
	go ln.Accept()
	fmt.Println("If you want to leave press sth: ")
	fmt.Scanf("%s",&leave)
	//client.ServeHttpAPI()
}

func (client clientInfo) Register(easy *curl.CURL, register string, IpToSend string){

	easy.Setopt(curl.OPT_POST,true)
	easy.Setopt(curl.OPT_VERBOSE, true)
	easy.Setopt(curl.OPT_URL,IpToSend+"/"+register+","+client.Id+","+client.myIp)
	easy.Setopt(curl.OPT_READFUNCTION,

		func(ptr []byte, userdata interface{}) int {
			// WARNING: never use append()
			if !sent {
				sent = true
				ret := copy(ptr, REQUEST_GROUPS)
				return ret
			}
			return 0
		})
	easy.Setopt(curl.OPT_POSTFIELDSIZE, len(REQUEST_GROUPS))
	if err := easy.Perform(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}
}

func Request(easy *curl.CURL,kind string,group string,IpToSend string) {

	easy.Setopt(curl.OPT_HTTPGET,true)
	easy.Setopt(curl.OPT_VERBOSE, true)
	if strings.Compare(kind,"group") == 0{
	    easy.Setopt(curl.OPT_URL,IpToSend+"/groups")
	    easy.Setopt(curl.OPT_READFUNCTION,

		func(ptr []byte, userdata interface{}) int {
			// WARNING: never use append()
			if !sent {
				sent = true
				ret := copy(ptr, REQUEST_GROUPS)
				return ret
			}
			return 0
		})
	  easy.Setopt(curl.OPT_POSTFIELDSIZE, len(REQUEST_GROUPS))
	}else {
		parameter := REQUEST_MEMBERS + ","+group
		easy.Setopt(curl.OPT_URL,IpToSend+"/members,"+group)
		easy.Setopt(curl.OPT_READFUNCTION,

			func(ptr []byte, userdata interface{}) int {
				// WARNING: never use append()
				if !sent {
					sent = true
					ret := copy(ptr, parameter)
					return ret
				}
				return 0
			})
		easy.Setopt(curl.OPT_POSTFIELDSIZE, len(parameter))
	}
	if err := easy.Perform(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}
}


func (clientInfo)NewClient(id, addr string) *clientInfo{
	cl := &clientInfo{myIp: addr, Id:id}
	return cl
}

func (client clientInfo) newListenerInterface(addr string) (*ListenerInterface, error) {
	fmt.Println("ListenerInterface!!!", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("----")
	}
	return &ListenerInterface{addr, ln.(*net.TCPListener)}, nil
}

func (s *ListenerInterface) Accept() {
	for{
		connIn, error := s.ln.Accept()
		if error != nil {
			if _, ok := error.(net.Error); ok {
				fmt.Println("Error received while listening.")
			}
		} else {
			var receivedMessage string
			json.NewDecoder(connIn).Decode(&receivedMessage)
			responseMessage := "Sure buddy.. too easy.. from " + s.address
			json.NewEncoder(connIn).Encode(&responseMessage)
			fmt.Println(receivedMessage)
		}
	}
}

// Close closes the service.
func (s *ListenerInterface) Close() {
	s.ln.Close()
	return
}

/*
func (m *clientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	switch {
	case r.Method == "POST":
		fmt.Println("Install client info")
		url, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}
		str := string(url)
		//str1 := strings.Split(str,",")
		fmt.Println("Client info..",string(key)+str)
		//m.kvstore.Add(string(key),str)

		//m.kvstore.Propose(string(key)+str)
		//TODO send info to group memb other nodes about new client
	}
	//fmt.Println(m.kvstore.ItemCount()) // just checking
}


func  (client clientInfo)ServeHttpAPI() {
	fmt.Println("Responding to Server...")
	srv := http.Server{
		Addr: client.myIp,
		Handler: &clientHandler{
			client: &(client),
		},
	}
	//go func() {
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
	///}()
	fmt.Println("HEREEEE")

	//TODO exit through channel or not?

}*/