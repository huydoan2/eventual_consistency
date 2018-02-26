# eventual_consistency

Names: Saharsh Oza and Huy Doan
UT EID: sso284 and hd5575


Key Ideas: 

1. Time: A vectorclock is used to maintain logical time through the system. The clock is updated as defined in Mattern '89
2. Total Order: The server describes a total order of the puts based on a combination of the vector clock and the client ID that performs the operation. Client ID breaks ties when two vector clocks corresponding to a put are concurrent.
3. Caching: 
a) Client Cache: 
	i) The protocol uses client side caching to provide the two session gaurantees of read your own write and montonic reads. 
	ii) The cache also reduces latency for every get operation. 
b) Server cache:
	i) Server side cache only stores put operations that occur in between 2 stabilize calls
	ii) Server side caching reduces latency of total ordering in the stabilize call. 
4. MST in Stabilize:
	On stabilize, the protocol must guarantee that every server sends its information to every other server. However, if done naively this can lead to increased network traffic and O(n^2) messages. To minimise this, we implement a gather-scatter algorithm that generates a MST with a random server as the root. 
5. Golang RPC is used for communication between processes.

Details of API Implementation:

1. Put: 
a) The master calls put on a client with a key-value pair. 
b) The client stores this entry in its cache
c) A server is selected at random from the list of servers connected to the client, and the a server side put is performed on it.
d) The server put causes it to update its cache and datastore

2. Get:
a) The master calls get on the client with a key
b) The client checks to see if its cache is valid. 
c) If it is, then it updates it replies to the master from its own cache. This will provide the two session guarantees.
d) If not, it will query a random server for the value. This will always return a value older than what the client has already cached for the following reason. The client cache will be invalid only after the servers stabilize. Hence, if the client queries the server, it is because a stabilize occurred among the servers. Stabilize will total order the puts for a key across all servers. This ensures that the new value it reads will be strictly more recent than its own cache.

3. Stabilize:
a) Master decides to pick a stabilize 