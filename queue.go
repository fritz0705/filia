package filia

// A CrawlerQueue is in most cases the same as a channel to send and receive
// strings, but provides two methods instead for external queue systems like
// Redis and RabbitMq
type CrawlerQueue interface {
	// Send sends the list of urls in given order to the queue
	Send(urls ...string)
	// Recv receives one url from the queue and returns it. It may block.
	Recv() (url string)
}

// StdCrawlerQueue is a string channel with methods required by CrawlerQueue
type StdCrawlerQueue chan string

// Send sends the urls to the string channel. It's just a wrapper for c <- url,
// but is needed to fulfill the CrawlerQueue interface.
func (s StdCrawlerQueue) Send(urls ...string) {
	for _, url := range urls {
		select {
		case s <- url:
		default:
			go func(){
				s <- url
			}()
		}
	}
}

// Recv receives one string and is just a wrapper for <-c, but is needed to
// fulfill the CrawlerQueue interface.
func (s StdCrawlerQueue) Recv() string {
	return <-s
}
