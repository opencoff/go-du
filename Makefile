
arch = $(shell ./build --print-arch)
bindir = ./bin/$(arch)
bin = $(bindir)/godu
installdir = $(HOME)/bin/$(arch)

all: $(bin)

.PHONY: $(bin)
$(bin):
	./build -s

install: $(bin)
	-cp -f $< $(installdir)/

.PHONY: clean

clean:
	-rm -rf ./bin
