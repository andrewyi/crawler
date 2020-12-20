package routingpool

type RoutingPool interface {
	Start() error
	Stop()
}
