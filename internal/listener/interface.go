package listener

type Listener interface {
	Listen(done <-chan bool) error
}
