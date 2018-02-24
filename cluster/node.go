package cluster

import ("strconv"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"net"
	"time"
	"encoding/json"
	"github.com/coreos/etcd/raft"
	"math"
	"golang.org/x/net/context"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/rafthttp"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/etcdserver/stats"
	"net/url"
	"net/http"
	"log"
	"bytes"
	"encoding/gob"
)



const hb = 1

type nodeInfo struct {
	nodeId     int    `json:"nodeId"`
	store *raft.MemoryStorage
	peers      []string
	groupIds   []string
	myIp       string
	myPort     string
	hashKey    string
	groupId    string
	kvstore    KVStore
	raft      raft.Node
	cfg      *raft.Config
	transport *rafthttp.Transport
	ctx context.Context
	ticker <-chan time.Time
	done <-chan struct{}
	proposeChannel <-chan string
	commitC chan<- *string // entries committed to log (k,v)
}

func NewNode(nodeId int, peers []string, myport string, proposeC chan string) (*nodeInfo) {
	store := raft.NewMemoryStorage()
	commitC := make(chan *string)
	var groupids []string
	groupids = make([]string,len(peers))
	for i,_ := range peers{
		groupids[i] = string('a'+i)
	}
	n := &nodeInfo{
		nodeId:	nodeId,
		peers:  peers,
		myIp: 	peers[nodeId-1],
		myPort:  myport,
		groupIds: groupids,
		groupId: groupids[nodeId-1],
		ctx:   context.TODO(),
		store: store,
		commitC: commitC,              // commits to KVStore
		proposeChannel:proposeC,     // proposals to raft
		cfg: &raft.Config{
			// ID in the etcd raft
			ID:              uint64(nodeId),  // random
			// ElectionTick controls the time before a new election starts
			ElectionTick:    10 * hb,
			// HeartbeatTick controls the time between heartbeats
			HeartbeatTick:   hb,
			// Storage contains the log
			Storage:         store,
			MaxSizePerMsg:   math.MaxUint16,
			MaxInflightMsgs: 256,
		},
		ticker: time.Tick(time.Second),
		done: make(chan struct{}),
		} //proposeC: proposeC,

	h := md5.New()
	str := strconv.Itoa(n.nodeId)+ n.myIp+n.groupId
	h.Write([]byte(str))
	n.hashKey = hex.EncodeToString(h.Sum(nil))
	n.kvstore = *newKVStore(proposeC, commitC, groupids)
	rpeers := make([]raft.Peer, len(peers))
	for i := range rpeers {
		rpeers[i] = raft.Peer{ID: uint64(i+1)}
	}
	n.raft = raft.StartNode(n.cfg, rpeers)
	// for comm between raft nodes!
	ss := &stats.ServerStats{}
	ss.Initialize()
	n.transport = &rafthttp.Transport{
		ID:          types.ID(n.nodeId),
		ClusterID:   0x1000,
		Raft:        n,
		ServerStats: ss,
		LeaderStats: stats.NewLeaderStats(strconv.Itoa(n.nodeId)),
		ErrorC:      make(chan error),
	}

	n.transport.Start()
	for i := range n.peers {
		if i+1 != n.nodeId {
			fmt.Println(types.ID(i+1),[]string{n.peers[i]})
			n.transport.AddPeer(types.ID(i+1), []string{n.peers[i]})
		}
	}
	go n.serveRaft()
	go n.run()
	go n.checkIfClientsAreRunning()
	return n
}

func (n *nodeInfo) GetRaft() (raft.Node){
	return n.raft
}

func (n *nodeInfo) run() {
	n.ticker = time.Tick(time.Second)
	go n.proposeValuesToCluster()
	for {
		select {
		case <-n.ticker:
			n.raft.Tick()
		case rd := <-n.raft.Ready():
			n.saveToStorage(rd.HardState, rd.Entries, rd.Snapshot)
			n.send(rd.Messages)
			if !raft.IsEmptySnap(rd.Snapshot) {
				n.processSnapshot(rd.Snapshot)
			}
			for _, entry := range rd.CommittedEntries {     // committed entries process them one by one
				n.process(entry)                                             // works as commit Channel
				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					cc.Unmarshal(entry.Data)
					n.raft.ApplyConfChange(cc)
				}
			}
			n.raft.Advance()
		case <-n.done:
			fmt.Println("EDWWW1111")
			return
		case err := <-n.transport.ErrorC:
			fmt.Println("ERRRORRRR:",err)
			return
		}
	}
}

func (n *nodeInfo) proposeValuesToCluster(){
	for n.proposeChannel != nil{
		prop, ok := <- n.proposeChannel
		if !ok {
			n.proposeChannel = nil
		} else {
			log.Println("Proposing value: ")
			// prop = key,value
			// blocks until accepted by raft state machine
			n.raft.Propose(context.TODO(), []byte(prop))
		}
	}
}

func (n *nodeInfo) process(entry raftpb.Entry) {

	if entry.Type == raftpb.EntryNormal && entry.Data != nil {
		//fmt.Println("normal message:", string(entry.Data))
		s := string(entry.Data)
		n.commitC <- &s
		fmt.Println("Message received.")
	}
}

func (n *nodeInfo) Process(ctx context.Context, m raftpb.Message) error {
	return n.raft.Step(ctx, m)
}


func (n *nodeInfo) processSnapshot(snapshot raftpb.Snapshot) {
	fmt.Println("Applying snapshot on node "+n.hashKey)
	n.store.ApplySnapshot(snapshot)
}

func (n *nodeInfo) saveToStorage(hardState raftpb.HardState, entries []raftpb.Entry, snapshot raftpb.Snapshot) {
	n.store.Append(entries)
	if !raft.IsEmptyHardState(hardState) {
		n.store.SetHardState(hardState)
	}
	if !raft.IsEmptySnap(snapshot) {
		n.store.ApplySnapshot(snapshot)
	}
}


func (n *nodeInfo) send(messages []raftpb.Message) {
	//for _, m := range messages {
	//	//fmt.Println("RAFTTTT>>>>",raft.DescribeMessage(m, nil))
	//	// send message to other node
	//	//nodes[int(m.To)].receive(n.ctx, m)
	//}
	n.transport.Send(messages)
}

func (node nodeInfo) String() string {
	return "NodeInfo:{ nodeId:" + strconv.Itoa(node.nodeId) + " myIP: "+ node.myIp + ", port:" + node.myPort + " hash: "+node.hashKey+ "+ }"
}

// To be removed
/*func (node nodeInfo) StartNode(){
	fmt.Println("MESA:   ",node.myIp)
	ln, err := newListenerInterface(node.myIp)
	//node.checkIfOthersExist(node.peers)
	if err == nil {
		ln.Accept()
		//if err == nil{
		//   fmt.Println(conn)
		//}
	}
}*/

func (node nodeInfo) GetIp() string{
	return node.myIp
}

func (n *nodeInfo) serveRaft() {
	url, err := url.Parse(n.peers[n.nodeId-1])
	if err != nil {
		log.Fatalf("raftexample: Failed parsing URL (%v)", err)
	}

	ln, err := newListenerInterface(url.Host)
	if err != nil {
		log.Fatalf("raftexample: Failed to listen rafthttp (%v)", err)
	}

	err = (&http.Server{Handler: n.transport.Handler()}).Serve(ln.ln)
	if err != nil{
		log.Fatalf("raftexample: Failed to serve rafthttp (%v)", err)
	}
	//select {
	//case <-rc.httpstopc:
	//default:
	//	log.Fatalf("raftexample: Failed to serve rafthttp (%v)", err)
	//}
	//close(rc.httpdonec)
}


func(n nodeInfo) checkIfClientsAreRunning(){
	for{
		<-time.After(5 * time.Second)
		n.clientsAlive(n.kvstore.store[n.groupId])
	}
}

func (node nodeInfo) clientsAlive(clients map[string]string) {

	for clientId, clientIP := range clients {
		url, err1 := url.Parse(clientIP)
		if err1 != nil {
			log.Fatalf("raftexample: Failed parsing URL (%v)", err1)
		}
		_, err := net.Dial("tcp", url.Host)
		if err != nil {
			if _, ok := err.(net.Error); ok {
				fmt.Println("Couldn't connect to client ", clientIP)
				//node.kvstore.removeEntry(key)b := [2]string{"Penn", "Teller"}
				prop := "REMOVE:"+node.groupId+","+clientId+","+clientIP
				var buf bytes.Buffer
				if err := gob.NewEncoder(&buf).Encode(prop); err != nil {
					log.Fatal(err)
				}
				node.raft.Propose(context.TODO(), []byte(string(buf.Bytes())))
			}

		}
	}
}

func (node nodeInfo) checkIfOthersExist(cluster[] string){
	for i:=0; i< len(cluster); i++ {
		//ip := strings.Split(cluster[i], ":")
		if strings.Compare(node.myIp, cluster[i]) != 0 {
			connOut, err := net.DialTimeout("tcp", cluster[i], time.Duration(10)*time.Second)
			if err != nil {
				if _, ok := err.(net.Error); ok {
					fmt.Println("Couldn't connect to cluster.", cluster[i])
				}
			}else{
				fmt.Println(node.myIp," Can Connect to peer: ",cluster[i])
				request := "Add me to cluster "+node.myIp
				json.NewEncoder(connOut).Encode(&request)
				var response string
				json.NewDecoder(connOut).Decode(&response)
				fmt.Println(response)
			}
			if(connOut != nil){
				connOut.Close()
			}
		}
	}
}

func (n *nodeInfo) IsIDRemoved(id uint64) bool { return false }
func (n *nodeInfo) ReportUnreachable(id uint64)                          {}
func (n *nodeInfo) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}