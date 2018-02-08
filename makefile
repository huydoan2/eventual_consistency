PACKAGE	= project1
GOPATH	= $(CURDIR)/.gopath
BASE	= $(GOPATH)/src/$(PACKAGE)
GO		= go

$(BASE):
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

.PHONY: all
all: | $(BASE)
	echo cd $(BASE) && $(GO) build -o bin/$(PACKAGE)