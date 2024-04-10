module github.com/opencoff/go-du

go 1.21.1

require (
	github.com/opencoff/go-utils v0.9.3
	github.com/opencoff/go-walk v0.6.1
	github.com/opencoff/pflag v1.0.6-sh2
)

require (
	github.com/opencoff/go-mmap v0.1.2 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.15.0 // indirect
)

// local testing
//replace github.com/opencoff/go-walk => ../go-walk
