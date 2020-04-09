// du.go - parallel du(1)
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
	"runtime"
	"sync"
)

type result struct {
	isdir bool
	name  string
	size  uint64 // will be zero for dirs
}

const (
	_Chansize          int = 65536
	_ParallelismFactor int = 2
)

type duState struct {
	followSymlinks bool
	all            bool
	ch             chan string
	out            chan result
	errch          chan error
	wg             sync.WaitGroup
}

func Walk(nm []string, all bool, followSymlinks bool) (chan result, chan error) {

	// number of workers
	nworkers := runtime.NumCPU() * _ParallelismFactor
	//nworkers = 4

	d := &duState{
		ch:             make(chan string, _Chansize),
		out:            make(chan result, _Chansize),
		errch:          make(chan error, 8),
		followSymlinks: followSymlinks,
		all:            all,
	}

	// start workers
	for i := 0; i < nworkers; i++ {
		go d.worker()
	}

	// send work to workers
	for i := range nm {
		d.wg.Add(1)
		d.ch <- nm[i]
	}

	// close the channels when we're all done
	go func() {
		d.wg.Wait()
		close(d.out)
		close(d.errch)
		close(d.ch)
	}()

	return d.out, d.errch
}

// worker thread to walk directories
func (d *duState) worker() {
	for nm := range d.ch {
		dirs, sz, err := d.walkPath(nm)

		if err != nil {
			d.errch <- err
			d.wg.Done()
			continue
		}
		d.out <- result{
			isdir: true,
			name:  nm,
			size:  sz,
		}

		// requeue the dirs
		d.wg.Add(len(dirs))
		for i := range dirs {
			d.ch <- dirs[i]
		}

		d.wg.Done()
	}
}

// process a directory and return the list of subdirs and a total of all regular
// file sizes
func (d *duState) walkPath(nm string) (dirs []string, tot uint64, err error) {
	var fi os.FileInfo

	if d.followSymlinks {
		fi, err = os.Stat(nm)
	} else {
		fi, err = os.Lstat(nm)
	}
	if err != nil {
		return nil, 0, err
	}

	m := fi.Mode()
	switch {
	case m.IsDir():
		// process it below

	case m.IsRegular():
		if d.all {
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
			}
			return nil, 0, err
		}
		return nil, uint64(fi.Size()), nil

	case (m & os.ModeSymlink) > 0:
		if d.all {
			if d.followSymlinks {
				fi, err = os.Stat(nm)
				if err != nil {
					return nil, 0, err
				}
			}
			tot += uint64(fi.Size())
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
			}
		}

	default:
		return nil, 0, err
	}

	fd, err := os.Open(nm)
	if err != nil {
		return nil, 0, err
	}

	fiv, err := fd.Readdir(-1)
	if err != nil {
		return nil, 0, err
	}

	dirs = make([]string, 0, len(fiv)/2)
	for i := range fiv {
		fi = fiv[i]
		m = fi.Mode()

		// we don't want to use filepath.Join() because it "cleans"
		// the path (removes the leading .)
		fp := fmt.Sprintf("%s/%s", nm, fi.Name())

		switch {
		case m.IsDir():
			dirs = append(dirs, fp)

		case m.IsRegular():
			tot += uint64(fi.Size())
			if d.all {
				d.out <- result{
					isdir: false,
					name:  fp,
					size:  uint64(fi.Size()),
				}
			}

		case (m & os.ModeSymlink) > 0:
			if d.all {
				if d.followSymlinks {
					fi, err = os.Stat(fp)
					if err != nil {
						return nil, 0, err
					}
				}
				tot += uint64(fi.Size())
				d.out <- result{
					isdir: false,
					name:  fp,
					size:  uint64(fi.Size()),
				}
			}
		default:
		}
	}
	fd.Close()

	return dirs, tot, nil
}
