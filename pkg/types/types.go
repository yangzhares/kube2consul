package types


// Service is to indicate a k8s pod record
// for headless services, will add a DNS record into consul
type Service struct {
	// Pod name as Node
	Node string
	// Address is Pod address
	Address string
	//  ID is Pod Name joining Address
	ID string
	// Name is service name joining namespace
	Name string

	Tags []string
	Port int
}