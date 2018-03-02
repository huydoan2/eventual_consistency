
ROOT = $(GOPATH)/src/github.com/huydoan2/eventual_consistency

.PHONY: all
all: server client master

.PHONY: server
server: vectorclock cache
	cd $(ROOT)/server;	go install

.PHONY: client
client: vectorclock cache
	cd $(ROOT)/client;	go install

.PHONY: master
master:
	cd $(ROOT)/master;	go install 


.PHONY: vectorclock 
vectorclock:
	cd $(ROOT)/vectorclock;	go install 

.PHONY: cache
cache: vectorclock
	cd $(ROOT)/cache;	go install

.PHONY: run
run: master
	cd $(GOPATH)/bin; ./master


.PHONY: clean
clean:
	rm -rf $(GOPATH)/bin/client \
	$(GOPATH)/bin/server \
	$(GOPATH)/bin/master \
	$(GOPATH)/bin/log/* \
	$(GOPATH)/pkg/linux_amd64/github.com/huydoan2/eventual_consistency/vectorclock.a \
	$(GOPATH)/pkg/linux_amd64/github.com/huydoan2/eventual_consistency/cache.a