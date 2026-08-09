module github.com/aixgo-dev/aixgate

go 1.26

// The single Go pin: CI reads this file via go-version-file, so bumping
// the patch here bumps it everywhere.
toolchain go1.26.5

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)
