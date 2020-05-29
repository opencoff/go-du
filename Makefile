
bindir = ./bin/$(shell ./build --print-arch)
bin = $(bindir)/godu
installdir = $(HOME)/bin/$(shell uname)

all: $(bin)

$(bin): main.go humansize.go die.go
	./build -s

install: $(bin)
	-cp -f $< $(installdir)/

.PHONY: clean

clean:
	-rm -rf ./bin
