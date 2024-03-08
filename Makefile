# kubefasthdfs
#

RM := $(shell [ -x /bin/rm ] && echo "/bin/rm" || echo "/usr/bin/rm" )
GOMOD=on
default: all

phony := all
all: build

phony += build server
build: server

server: 
	@build/build.sh server $(GOMOD)

phony += clean
clean:
	@$(RM) -rf build/bin

.PHONY: $(phony)
