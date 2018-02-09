ROOT = $(CURDIR)

.PHONY: all
all: server client master

.PHONY: server
server:
	cd $(ROOT)/server;	go build
	cd ..

.PHONY: client
client:
	cd $(ROOT)/client;	go build
	cd ..

.PHONY: master
master:
	cd $(ROOT)/master;	go build
	cd ..

.PHONY: run
run:
	$(ROOT)/master/master

.PHONY: clean
clean:
	rm -rf $(ROOT)/client/client $(ROOT)/server/server $(ROOT)/master/master $(ROOT)/log/*