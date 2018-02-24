package cluster

import (
	"sync"
	"bytes"
	"encoding/gob"
	"log"
	"strings"
	"fmt"
)

type KVStore struct {
	//	proposeC chan<- string // channel for proposing updates
	mu    sync.RWMutex
	store map[string]map[string]string  // current committed group - node pairs
	proposeChannel chan<- string
}

//var m = make(map[string]map[string]int)
//m["test"] = make(map[string]int)
//m["test"]["test"] = 1

func newKVStore(propChannel chan<- string, commitC <-chan *string, groupids []string) *KVStore {

	s := &KVStore{
		proposeChannel:propChannel,
	}
	s.store = make(map[string]map[string]string)
	for _, v := range groupids {
		s.store[v] = make(map[string]string)
	}
	go s.readCommitChannel(commitC)
	return s
}

func (kv *KVStore) Lookup(a, key string) (string, bool) {
	kv.mu.RLock()
	v, ok := kv.store[a][key]
	defer kv.mu.RUnlock()
	return v, ok
}

func (kv *KVStore) ListMembersOfGroup(a string) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	for i, j := range kv.store[a]{
		fmt.Println(i+" -> "+j)
	}
}

func (kv *KVStore) Propose(proposal string){
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(proposal); err != nil {
		log.Fatal(err)
	}
	kv.proposeChannel <- string(buf.Bytes())
}

func (kv *KVStore) Add(group, k, v string) {
	kv.mu.Lock()
	child, ok := kv.store[group]
	if !ok {
		child = map[string]string{}
		kv.store[group] = child
	}
	child[k] = v
	kv.mu.Unlock()
}

func (kv *KVStore) readCommitChannel(commit <-chan *string){
	for data := range commit {
		if data != nil{
			var receivedMessage string
			dec := gob.NewDecoder(bytes.NewBufferString(*data))//NewBuffer(entry.Data))
			if err := dec.Decode(&receivedMessage); err != nil {
				log.Fatalf("raftexample: could not decode message (%v)", err)
			}
			if len(receivedMessage) > 1 {
				entry := strings.Split(receivedMessage,":")
				if strings.Compare(entry[0],"REMOVE") == 0 {
					kv.removeEntry(entry[1])
					fmt.Println("Remove entry: " + entry[1])
				}
				param := strings.Split(receivedMessage, ",")
				kv.Add(param[0], param[1], param[2])
			}
		}
	}
}

func (kv *KVStore) removeEntry(removal string){
	param := strings.Split(removal, ",")
	fmt.Println("Deletes: ",param[0]," ",param[1])
	delete(kv.store[param[0]],param[1])
}

func (kv *KVStore) ItemCount() int {
	kv.mu.RLock()
	n := len(kv.store)
	kv.mu.RUnlock()
	return n
}
