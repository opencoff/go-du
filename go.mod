module github.com/opencoff/go-du

go 1.20

require (
	github.com/opencoff/go-utils v0.8.0
	github.com/opencoff/go-walk v0.5.0
	github.com/opencoff/pflag v1.0.6-sh2
)

require (
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
)

// local testing
//replace github.com/opencoff/go-walk => ../go-walk
