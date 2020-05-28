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
	"sync"
	"syscall"
	"runtime"
)

type result struct {
	isdir bool
	name  string
	size  uint64 // will be zero for dirs
	stat  *syscall.Stat_t
}

const (
	_Chansize          int = 65536
	_ParallelismFactor int = 2
)

type duState struct {
	followSymlinks bool
	all            bool
	oneFS          bool
	ch             chan string
	out            chan result
	errch          chan error
	wg             sync.WaitGroup

	// track device maj:min to stay within a single filesys
	fs sync.Map

	// track hardlinked files and count only once in a subtree
	hardlink sync.Map
}

func Walk(names []string, all, oneFS, followSymlinks bool) (chan result, chan error) {

	// number of workers
	nworkers := runtime.NumCPU() * _ParallelismFactor

	d := &duState{
		followSymlinks: followSymlinks,
		all:            all,
		oneFS:          oneFS,
		ch:             make(chan string, _Chansize),
		out:            make(chan result, _Chansize),
		errch:          make(chan error, 8),
	}

	// start workers
	for i := 0; i < nworkers; i++ {
		go d.worker()
	}

	// send work to workers
	for i := range names {
		var fi os.FileInfo
		var err error

		nm := names[i]
		if d.followSymlinks {
			fi, err = os.Stat(nm)
		} else {
			fi, err = os.Lstat(nm)
		}
		if err != nil {
			d.errch <- err
			continue
		}

		m := fi.Mode()
		switch {
		case m.IsDir():
			// we only give dirs to workers
			if oneFS {
				d.TrackFS(nm, fi)
			}
			d.wg.Add(1)
			d.ch <- nm

		case m.IsRegular():
			if ino, n := nlinks(fi); n > 1 {
				if d.TrackInode(ino, nm) {
					continue
				}
			}
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
				stat:  fi.Sys().(*syscall.Stat_t),
			}

		case (m & os.ModeSymlink) > 0:
			if d.followSymlinks {
				fi, err = os.Stat(nm)
				if err != nil {
					d.errch <- err
				}
			}

			if !d.IsOneFS(nm, fi) {
				continue
			}

			if ino, n := nlinks(fi); n > 1 {
				if d.TrackInode(ino, nm) {
					continue
				}
			}
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
				stat:  fi.Sys().(*syscall.Stat_t),
			}

		default:
		}
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
		if !d.IsOneFS(nm, fi) {
			return nil, 0, nil
		}
		// process regular dirs below

	case m.IsRegular():
		if !d.IsOneFS(nm, fi) {
			return nil, 0, nil
		}

		if ino, n := nlinks(fi); n > 1 {
			if d.TrackInode(ino, nm) {
				return nil, 0, nil
			}
		}

		if d.all {
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
				stat:  fi.Sys().(*syscall.Stat_t),
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

			if !d.IsOneFS(nm, fi) {
				return nil, 0, nil
			}

			if ino, n := nlinks(fi); n > 1 {
				if d.TrackInode(ino, nm) {
					return nil, 0, nil
				}
			}

			tot += uint64(fi.Size())
			d.out <- result{
				isdir: false,
				name:  nm,
				size:  uint64(fi.Size()),
				stat:  fi.Sys().(*syscall.Stat_t),
			}
		}
		return nil, uint64(fi.Size()), nil

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
			if !d.IsOneFS(fp, fi) {
				continue
			}
			dirs = append(dirs, fp)

		case m.IsRegular():
			if !d.IsOneFS(fp, fi) {
				continue
			}

			if ino, n := nlinks(fi); n > 1 {
				if d.TrackInode(ino, fp) {
					continue
				}
			}

			tot += uint64(fi.Size())
			if d.all {
				d.out <- result{
					isdir: false,
					name:  fp,
					size:  uint64(fi.Size()),
					stat:  fi.Sys().(*syscall.Stat_t),
				}
			}

		case (m & os.ModeSymlink) > 0:
			if d.all {
				if !d.IsOneFS(nm, fi) {
					continue
				}

				if d.followSymlinks {
					fi, err = os.Stat(fp)
					if err != nil {
						return nil, 0, err
					}
				}
				if ino, n := nlinks(fi); n > 1 {
					if d.TrackInode(ino, nm) {
						continue
					}
				}
				tot += uint64(fi.Size())
				d.out <- result{
					isdir: false,
					name:  fp,
					size:  uint64(fi.Size()),
					stat:  fi.Sys().(*syscall.Stat_t),
				}
			}
		default:
		}
	}
	fd.Close()

	return dirs, tot, nil
}

// return true if the file 'nm' is on the same filesystem as the command line args
func (d *duState) IsOneFS(nm string, fi os.FileInfo) bool {
	if !d.oneFS {
		return true
	}

	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		if _, ok := d.fs.Load(st.Dev); ok {
			return false
		}
	}
	return true
}

// track the name and the device major/minor against it
func (d *duState) TrackFS(nm string, fi os.FileInfo) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		d.fs.Store(st.Dev, nm)
	}
}

// track the inode and filename for hardlinks
func (d *duState) TrackInode(ino uint64, nm string) bool {
	//_, file, line, _ := runtime.Caller(1)
	//fmt.Printf("inode %d: %s - caller %s:%d\n", ino, nm, file, line)
	if _, ok := d.hardlink.LoadOrStore(ino, nm); ok {
		//fmt.Printf("inode %d: already tracked as %s; skipping %s\n", ino, v.(string), nm)
		return true
	}

	//fmt.Printf("inode %d: +tracked (%s)\n", ino, nm)
	return false
}


// return inode and number of hardlinks
func nlinks(fi os.FileInfo) (inode uint64, nlinks uint64) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		inode = st.Ino
		nlinks = st.Nlink
	}

	return
}
