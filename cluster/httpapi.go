package cluster

import (
	"net/http"
	"fmt"
	"log"
	"io/ioutil"
	"strings"
	"math/rand"
	"time"
)

type myHandler struct{
	node    *nodeInfo
	proposeC   <-chan(string)
}

func (m *myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	switch {
	case r.Method == "POST":
		ran := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		fmt.Println("Install client info")
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}
		//str := string(url)
		//str1 := strings.Split(str,",")
		//fmt.Println("Client info..",string(key)+" ... "+str)
		s := strings.TrimPrefix(string(key),"/")
		gro := ran.Int31n(10)%3
		fmt.Println("S: "+s)
		prop := m.node.groupIds[gro] + s[1:len(s)]
		fmt.Println("Prop: "+prop)
		m.node.kvstore.Propose(prop)
	case r.Method == "GET":
		fmt.Println("Process client request")
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}
		//str := string(url)
		//fmt.Println("Client info..",string(key),"   ...  ",str)
		parameter := strings.TrimPrefix(string(key),"/")
		if strings.Compare(parameter,"groups") == 0{
			for _, arg := range  m.node.groupIds{
				fmt.Printf("Group Name: %s\n",arg)
			}
		}else{
			group := strings.Split(string(key),",")[1]
			m.node.kvstore.ListMembersOfGroup(group)
		}
	}
	//fmt.Println(m.kvstore.ItemCount()) // just checking
}

func (node nodeInfo) ServeHttpKVAPI() {
	fmt.Println("Starting Serving Clients...")
	srv := http.Server{
		Addr:    ":" + node.myPort,
		Handler: &myHandler{
			node:	&(node),
		},
	}
	//go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	///}()
	//TODO exit through channel or not?

}