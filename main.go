package main

import (
	"flag"
	"strings"
	"groupmembership_test/cluster"
	"fmt"
)


func main(){
	nodeId := flag.Int("id", 1, "ID")
  	peers := flag.String("cluster", "http://127.0.0.1:9021", "comma separated cluster peers")
	myport := flag.String("port", "8001", "ip address to run this node on. default is 8001.")
	//groupId := flag.String("group", "a", "GroupIds comma separated")
	flag.Parse()
	fmt.Println("STARTS!!!")
	proposeChannel := make(chan string)
	defer close(proposeChannel)

	node := cluster.NewNode(*nodeId, strings.Split(*peers,","),*myport,proposeChannel)
	fmt.Println(node.String())

	// Wait for leader, is there a better way to do this
	//for node.GetRaft().Status().Lead != 1 {
	//	time.Sleep(100 * time.Millisecond)
	//}
	fmt.Println("HEREEEE")
	node.ServeHttpKVAPI()
}