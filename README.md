# pond | forked from github.com/alitto/pond

- minimalistic and high-performance goroutine worker pool written in Go
- forked from github.com/alitto/pond for the express purpose of adding 1 function:
  - func group.submitWithArgs(task func(args map[string]interface{}) error, args map[string]interface{})
- full API reference is available at https://pkg.go.dev/github.com/alitto/pond