[![GoDoc](https://godoc.org/github.com/opencoff/go-du?status.svg)](https://godoc.org/github.com/opencoff/go-du)

# README for go-du
This repository is **ARCHIVED**; its content is moved to
[go-progs](https://github.com/opencoff/go-progs).


## What is this?
`go-du` is an opionated re-implementation of du(1):

* it traverses all directories in parallel: bounded by the
  concurrency offered by the CPU
* it optionally prints all files in dirs/subdirs
* it prints a total
* it sorts the output by largest-size before printing


## How do I build it?
With Go 1.5 and later:

    git clone https://github.com/opencoff/go-du
    cd go-du
    make

The binary will be in `./bin/$HOSTOS-$ARCH/godu`.
where `$HOSTOS` is the host OS where you are building (e.g., openbsd)
and `$ARCH` is the CPU architecture (e.g., amd64).

## How do I use it?
Examples:

    # traverse dirs on command line
    ./bin/linux-amd64/godu -h *

    # print all files
    ./bin/linux-amd64/godu -h -a *


## Licensing Terms
The tool and code is licensed under the terms of the
GNU Public License v2.0 (strictly v2.0). If you need a commercial
license or a different license, please get in touch with me.

See the file ``LICENSE.md`` for the full terms of the license.

## Author
Sudhi Herle <sw@herle.net>
