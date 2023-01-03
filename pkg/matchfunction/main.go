package main

import (
	"matchmaker/pkg/matchfunction/mmf"
)

const (
	queryServiceAddress = "open-match-query.open-match.svc:50503" // Address of the QueryService endpoint.
	serverPort          = 50502                                   // The port for hosting the Match Function.
)

func main() {
	mmf.Start(queryServiceAddress, serverPort)
}
