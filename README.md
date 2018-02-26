# eventual_consistency

Names: Saharsh Oza and Huy Doan
UT EID: sso284 and hd5575


Key Ideas: 

1. Time: A vectorclock is used to maintain logical time through the system. The clock is updated as defined in Mattern '89. During any interaction between client and server, their vector clocks are synchronized.
2. Total Order: The server describes a total order based on a combination of the vector clock and the ID that performs the operation. ID breaks ties when two vector clocks are concurrent.
3. Caching: 
a) Client Cache (write through): 
	i) The protocol uses client side caching to provide the two session gaurantees of read your own write and montonic reads. 
	ii) The cache also reduces latency for every get operation. 
	iii) Client cache manages its invalidation by querying a version number from the server. This ensures it does not return stale data after a stabilize
b) Server cache:
	i) Server side cache only stores Put operations that occur in between 2 stabilize calls
	ii) Server side caching reduces latency of total ordering in the stabilize call. 
4. MST in Stabilize:
	On stabilize, the protocol must guarantee that every server sends its information to every other server. However, if done naively this can lead to increased network traffic and O(n^2) messages. To minimise this, we implement a gather-scatter algorithm that generates a MST with a random server as the root. 
5. Golang RPC is used for communication between processes.

Details of API Implementation:

1. Put [clientID] [key] [value]: 
a) The master calls Put on a client with a key-value pair. 
b) Client Put: 
	i) Client picks a random server then querries the version of the server. If the client's version number is outdated, it invalidates its cache. This means that servers are stabilized and have the most updated version and the client is outdated.
	ii) The client stores the k-v-time entry in its cache. Time is the current client vector clock
	iii) Client calls Put RPC on the chosen server from step(i).
c) Server Put:
	i) The server syncs its time with the client time in the request
	ii) The server first checks if the entry exists in its data store and has a higher timestamp that the client request. In that case, the server ignores the client request.
	iii) The server adds the k-v-time entry to its cache and data store. Time is the time in the client's request.
	iv) Server sends back its incremented time and the entry in its data store to the client.
d) Client after Server Put RPC returns:
	i) Synchronizes its time with that in the server response
	ii) Update its cache if server response with an entry which has higher time value

2. Get [clientID] [key]:
a) The master calls Get on the client with a key
b) Client Get:
	i) The client check for cace invalidation as in Put (b.i)
	ii) Check to see if the key is present in the cache. If yes, return it to the master. Else query a random server for it.
c) Server Get:
	i) The server syncrhonizes its time with client's request. It then tries to return the entry in its data store for the queried key.
	This will never return a value older than what the client has already cached for the following reason. The client cache will be invalid only after the servers stabilize. Hence, if the client queries the server, it is because a stabilize occurred among the servers. Stabilize will total order the puts for a key across all servers. This ensures that the new value it reads will be strictly more recent than its own cache.
	ii) If the server does not have the key in its data store, it returns "ERR_KEY" instead.
	iii) The server includes its incremented time in the response to the client.
d) Client after Server RPC Get returns:
	i) Syncrhonizes its time and cache entry according to the server's response. Similar to Put

3. Stabilize:
a) Master runs stabilize on all parititions
b) Master picks a random server and initiates stabilize on it. This server plays as the root of the MST for its partition. The root then calls stabilize on its self to start the process. A stabilize call on a server follows the steps: Gather, and Scatter
c) Gather (Converge cast):
	i) The node checks to see if it already has a parent. If it does, it returns. Note that all but one RPC call will return in this manner. A node puts itself to the MST by not responding immediately to the caller.
	ii) Else, the node keeps spanning the Gather call on other servers that it has connection with. If the node is a leaf of the MST, it sends its whole cache and time to its parent. Sending only the cache to the parent significantly reduces the traffic over the network.
	iii) The node gathers the cache from its children and merge them to its cache. It also synchronizes its time with all of the children. From this step, a node knows about its children and it's important for Scatter.
	iv) The node then replies to the parent its cache content and time.
d) The root orders the entries it received and calls scatter. The cache sent to the children holds the final total ordered content
e) Scatter (Broadcast):
	i) The node updates its data store and time with the content sent by the parent. 
	ii) The cache is then invalidated.
	iii) The node spans Scatter calls on its children only. This keeps the traffic minimum by not sending the entire cache content from the parent to everyone the node is connected to.
	iv) At the end of the scatter, a version number is updated. This lets the client know whether a new stabilize has been called, in which case it will know that its client side cache may be stale.