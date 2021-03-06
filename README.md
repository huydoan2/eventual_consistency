# eventual_consistency

Names: Saharsh Oza and Huy Doan
UT EIDs: sso284 and hd5575

The project implements a distributed key-value store system with eventual consistency. The language used is Go. The structure is as follow: the master program is used to create servers, clients, connections, and read/write data. The master, in brief, simulates the execution of the systems.

## Key Ideas: 

1. Time: A vectorclock is used to maintain logical time through the system. The clock is updated as defined in Mattern '89. During any interaction between client and server, their vector clocks are synchronized.
2. Total Order: The server describes a total order based on a combination of the vector clock and the ID that performs the operation. ID breaks ties when two vector clocks are concurrent.
3. Caching: 
a) Client Cache (write through): 
	i) The protocol uses client side caching to provide the two session gaurantees of read your own write and montonic reads. 
b) Server cache:
	i) Server side cache only stores Put operations that occur in between 2 stabilize calls
	ii) Server side caching reduces latency of total ordering in the stabilize call. 
4. MST in Stabilize:
	On stabilize, the protocol must guarantee that every server sends its information to every other server. However, if done naively this can lead to increased network traffic and O(n^2) messages. To minimise this, we implement a gather-scatter algorithm that generates a MST with a random server as the root. 
5. Golang RPC is used for communication between processes.

## Details of API Implementation:

1. put [clientID] [key] [value]: 
a) The master calls Put on a client with a key-value pair. 
b) Client Put: 
	i) The client stores the k-v-time entry in its cache. Time is the current client vector clock
	ii) Client calls Put RPC on the chosen server from step(i).
c) Server Put:
	i) The server syncs its time with the client time in the request
	ii) The server first checks if the entry exists in its data store and has a higher timestamp that the client request. In that case, the server ignores the client request.
	iii) The server adds the k-v-time entry to its cache and data store. Time is the time in the client's request.
	iv) Server sends back its incremented time and the entry in its data store to the client.
d) Client after Server Put RPC returns:
	i) Synchronizes its time with that in the server response
	ii) Update its cache if server response with an entry which has higher time value


2. get [clientID] [key]: The clieent querries a random server it connects to for the value of the key. If the server's response value has a stale value and client has newer value in its cache, it returns the cached value. Otherwise it updates its cache and returns server's response. Client synchronizes its time with the server through this process, too.

a) The master calls Get on the client with a key
b) Client Get:
	i) The client querries a random server
	ii) The client compare the server's response with what it has in the cache and respond appropriately
c) Server Get:
	i) The server syncrhonizes its time with client's request. It then tries to return the entry in its data store for the queried key.	
	ii) If the server does not have the key in its data store, it returns "ERR_KEY" instead.
	iii) The server includes its incremented time in the response to the client.
d) Client after Server RPC Get returns:
	i) Syncrhonizes its time and cache entry according to the server's response. Similar to Put
	ii) Compares the cache value and server's response. Update the cache accordingly.


3. stabilize: All servers in the same partition will have a uniform datastore after stabilizing. This property is not guaranteed for servers in different isolated partitions.

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


4. killServer [id]:
a) Master tells the target server to clean up: connections, close log file, etc.
b) Master send SIGKILL to the target server to actually kill the process.


5. joinServer [id]:
a) Master creates the server process and pass the id and the list of existing servers as command-line arguments.
b) The server process calls its Init() method to set up its state and connect to other servers. Once it connects to other servers as a client, it send RPCs to other servers and asked them to connect to it as clients. After this, the new server has bi-directional channels with all existing servers.


6. joinClient [clientId][serverId]:
a) Master creates the client process similarly to how it creates a server process.
b) The client connects to the target server socket.


7. createConnection [id1][id2]:
a) Master asks process with id1 to join process id2 as a client.
b) Process id1 then ask process id2 to join it as a client.


8. breakConnection [id1][id2]:
a) Master in parallel ask two processes to close the client connection to the other process.


9. printStore [id]:
a) Master ask the target server for its data store.
b) Master prints the data store out to Stdin.

10. test
a) The program enters a test mode. 
b) Inside test mode, "list" command will list all the available tests we provided and "list-desc" command will give a detailed description of each test.
c) From inside the test mode, any test can be executed by entering its name as presented in the "list" command.
d) Use the "exit" command to exit the test mode. Note that you cannot run api commands in the test mode.

## Performance:

There are 2 tests in the test suite of the project that test the performance of puts in the system. Time is measured after a combination of puts and stabilize. The tests are listed in `list` command in test mode; and are called `PerformanceTestSimple` and `PerformanceTestSingleServer`. Each performance test is done under 2 extreme settings. The first setting is that of 0 conflict (all clients put different keys) and the next with only conflict (all clients put the same key). These are referred to as "No conflict" and "Only conflict" respectively. In both of these, a stabilize call is made in the end. The measured time is the sum of time taken for 40 puts and a stabilize call.

`PerformanceTestSingleServer` connects 5 clients to a single server and issues 40 puts on the clients in a round robin manner. 
`PerformanceTestSimple` creates 5 clients, 5 servers and then connects a client to 1 server each.

The median time in msec over 5 runs in each of the 4 configurations is attached.

![Benchmark Plot](benchmark.png)

The following can be interpreted from the results:
1. System Configuration: Having a single server gives better performance than 5 servers. This can be attributed to the network overhead of stabilize. However, experiments reveal that this accounts for only ~50% of the overhead (due to the MST algorithm that minimises network overhead). Hence, another possibility could be that the experiment is run on a single node with 4 physical cores. Hence the configuration with 10 processes results in 
OS scheduling conflicts that slow it down as compared to the configuration with only 6 processes.
2. Put pattern: When puts that always conflict are applied to both the systems, better performance is observed than in a system with no conflict. While no detailed analysis has been performed, this could be attributed to increased memory allocation with puts of different keys vs a single memory over written by different puts.

### Note:
1. We did not mention the details of checking the validity of arguments and the state of the system such as whether that client/server exists. Look at the code for more details.
2. The master connects to all processes as a client so that it can make RPC requests to them.
3. The system can only accomodate 10 processes due to the limitation in vectorclock's implementation
4. Each process has a log in the "log" directory. Refer to them for more information, especially for debugging.
5. Building the project still has trouble with the two packages: vectorclock and cache. Please use the pre-built packages included in the directories.

Build and Run the project:
1. Make sure that you have a go workspace in $HOME/go which contains 3 directories: bin, pkg, and src
2. Make sure that your GOPATH is the default GOPATH ($HOME/go). Add the path to the bin directory to your PATH
3. Prepare the directory as $HOME/go/src/github.com/huydoan2 and extract the eventual_consistency directory inside of this directory
4. Go to the project's root folder ("eventual_consistency) and type make. This creates the binaries (client, server, and master) in the $HOME/go/bin directory and 2 ".a" packages (vectorclock and cache) in the $HOME/go/pkg/linux_amd64/github.com/huydoan2/eventual_consistency directory.
5. Make sure that the "log" directory exists in the $HOME/go/bin directory. If not, create one.
6. Go to the $HOME/go/bin directory.
7. run ./master < input.txt assuming the input.txt is the file containing the commands following the format (commands are case sensitive):
	command_api_1 arg1 arg2 arg3 ...
	command_api_2 arg1 ...

8. There are some tests already written as functions in the master program. Enable them in the main function to run the tests.

9. Run the "exit" command on the master command line prompt to safely close all of the processes and exit.