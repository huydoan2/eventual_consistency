ROOT = $(CURDIR)

.PHONY: all
all: server client master

.PHONY: server
server: cache vectorclock
	cd $(ROOT)/server;	go build
	cd ..

.PHONY: client
client: cache vectorclock
	cd $(ROOT)/client;	go build
	cd ..

.PHONY: master
master:
	cd $(ROOT)/master;	go build
	cd ..

.PHONY: cache
cache:
	cd $(ROOT)/cache;	go build
	cd ..

.PHONY: vectorclock
vectorclock:
	cd $(ROOT)/vectorclock;	go build
	cd ..

.PHONY: run
run:
	$(ROOT)/master/master

.PHONY: clean
clean:
	rm -rf $(ROOT)/client/client $(ROOT)/server/server $(ROOT)/master/master $(ROOT)/log/*