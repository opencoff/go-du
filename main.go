// main.go - parallel du(1)
//
// (c) 2016 Sudhi Herle <sudhi@herle.net>
//
// Licensing Terms: GPLv2
//
// If you need a commercial license for this work, please contact
// the author.
//
// This software does not come with any express or implied
// warranty; it is provided "as is". No claim  is made to its
// suitability for any purpose.

package main

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	flag "github.com/opencoff/pflag"
)

var Z string = path.Base(os.Args[0])
var Verbose bool

type rslice []result

func (r rslice) Len() int {
	return len(r)
}

func (r rslice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// we're doing reverse sort.
func (r rslice) Less(i, j int) bool {
	return r[i].size > r[j].size
}

func main() {
	var version bool
	var human bool
	var kb bool
	var byts bool
	var total bool
	var symlinks bool
	var all bool

	flag.BoolVarP(&version, "version", "", false, "Show version info and quit")
	flag.BoolVarP(&Verbose, "verbose", "v", false, "Show verbose output")
	flag.BoolVarP(&symlinks, "follow-symlinks", "L", false, "Follow symlinks")
	flag.BoolVarP(&all, "all", "a", false, "Show all files & dirs")
	flag.BoolVarP(&human, "human-size", "h", false, "Show size in human readable form")
	flag.BoolVarP(&kb, "kilo-byte", "k", false, "Show size in kilo bytes")
	flag.BoolVarP(&byts, "byte", "b", false, "Show size in bytes")
	flag.BoolVarP(&total, "total", "t", false, "Show total size")

	flag.Usage = func() {
		fmt.Printf(
			`%s - disk utilization calculator (parallel edition)

Usage: %s [options] dir [dir...]

Options:
`, Z, Z, Z)
		flag.PrintDefaults()
		os.Stdout.Sync()
		os.Exit(0)
	}

	flag.Parse()
	if version {
		fmt.Printf("%s - %s [%s; %s]\n", Z, ProductVersion, RepoVersion, Buildtime)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		die("Insufficient args. Try %s --help", Z)
	}

	var size func(uint64) string = humansize

	if human {
		size = humansize
	} else if kb {
		size = func(z uint64) string {
			z /= 1024
			return fmt.Sprintf("%d", z)
		}
	} else {
		size = func(z uint64) string {
			return fmt.Sprintf("%d", z)
		}
	}

	ch, ech := Walk(args, all, symlinks)

	// harvest errors
	errs := make([]string, 0, 8)
	go func() {
		for e := range ech {
			errs = append(errs, fmt.Sprintf("%s", e))
		}
	}()

	// now harvest dirs and files
	var tot uint64
	rv := make([]result, 0, 8192)
	rm := make(map[string]uint64)
	for r := range ch {
		if r.isdir {
			for i := range args {
				s := args[i]
				if strings.HasPrefix(r.name, s) {
					rm[s] += r.size
				}
			}
		} else {
			// must have come from the command line args _or_ it's a file
			tot += r.size
			rv = append(rv, r)
		}
	}

	if len(errs) > 0 {
		die("%s", strings.Join(errs, "\n"))
	}

	if !all {
		for k, v := range rm {
			tot += v
			rv = append(rv, result{name: k, size: v})
		}

	}
	sort.Sort(rslice(rv))
	for i := 0; i < len(rv); i++ {
		r := rv[i]
		fmt.Printf("%12s %s\n", size(r.size), r.name)
	}
	if total {
		fmt.Printf("%12s TOTAL\n", size(tot))
	}
}

// This will be filled in by "build"
var RepoVersion string = "UNDEFINED"
var Buildtime string = "UNDEFINED"
var ProductVersion string = "UNDEFINED"
