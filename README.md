# Implementation of Group Membership Service based on Raft


### Summary

This is a project for LPD (Distributed Programming Laboratory) of EPFL supervised by Dragos-Adrian Seredinschi. 

### Goal

The goal of the project was to design and implement a fault tolerant cluster that provides Group Membership Services using Go.
The cluster supports large-scale  storage services with high availability and consistency.
Following the lectures of the Distributed Algorithms course taught by Rachid Guerraoui we implemented and a service that accepts join and leave requests from processes explicitly and also provides information about cluster nodes and the processes each one is responsible for.

### Concept

One of the key aspects of our project was to have dynamic groups:


1) Replicated service across the cluster nodes

2) Clients can restart and rejoin the cluster

3) Client processes can crash, this is detected and they are removed from the group


###  Failure Detection

Failure Detection is managed and coordinated by each node of the GM cluster by sending heartbeats to the client processes it is responsible for.


### Architecture

The GM Service is reliable and scalable, since a small number of nodes (tested for 3,5) create a cluster coordinator of the GM Service. Large number of clients can join the GM Service and eventually they create groups on top of each cluster node.

All nodes of the GM cluster are updated on the clients of each group, so when they are requested to provide information about a group, every node is reliable to send out a consistent view of a specific group. 

In the project we integrated Raft Consensus protocol implemented by etcd to manage replication across the group membership cluster and we created a key-value store that holds information on all the groups's information that is committed and an HTTP Server that accepts REST requests.
```sh
map[map[string]]string
information is stored so as to give info about GM Service on each group:
group "a" : "nodeID1" -> "IP1:Port1"
             "nodeID2" -> "IP2:Port2"
             "nodeID3" -> "IP2:Port3"
```
The REST API exposes interfaces that shall be accessed from a client to get information for a specific group of the GM Cluster (GET command) and to join a group(POST command).

The clients (client.go) execute a curl command to learn the nodes that implement the GM Service, then they have the option to perform another curl command, to see the members of a specific group. At last, they choose the group identifier ('a','b' or 'c' if it is a 3-node cluster) to register to this group by executing another curl command and sending through it their clientId, their IP address and the port on which they listen().

Start a three node cluster, using the Procfile:
```sh
goreman start
```
Then to run the clients: 
```sh
go run client.go --id 1 --address http://127.0.0.1:9021 --clusterAddress http://127.0.0.1:22380
```