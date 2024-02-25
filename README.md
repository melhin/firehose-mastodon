# Firehose Mastodon

## Intro
A small utility demonstration for using channels and SSE together.
The application streams data from a single mastodon instance and then distributes it among the connected streams

## Notes
Testing out a separated manager that handles all the routing and distributing of incoming events.
The go routines then receive messages from channels based on which they act
