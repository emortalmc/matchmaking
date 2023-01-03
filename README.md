# Matchmaking

This is the core matchmaker for EmortalMC using OpenMatch and written in Golang.

Other notes:
  - We use backfills to create a match with few players and add more afterwards.
  - We use high density gameservers to run multiple games on a single GameServer.
  - There is not currently any party support.

## Components

### OpenMatch

OpenMatch handles the registering and lifecycle of tickets. The business logic is left to our components.

### Director

The director is responsible for pooling tickets together (although only an 'all' pool is used right now).
This would be used for having roles or MMR in the future.

The director also handles assigning servers to a match through Agones (GameServerAllocation) using the k8s API

### Matchmaking Function (MMF)

The MMF is responsible for taking the pool of tickets and creating matches from them.
If backfills are used, it is also responsible for creating these and filling them.

## Specifications

### Tickets

```
PersistentField:
  playerId:
    type: string
    description: The player's id
```

### Matches

```

```

### Backfills

```
Extensions:
  originalMatchId:
    type: string (of a UUID)
    description: The original match id the backfill was created for
```

### GameServer Data

```
Labels:
Annotations:
  openmatch.dev/match-id: {matchId} 
  openmatch.dev/expected-players: {jsonUuidArray}
  openmatch.dev/backfill-id: {backfillId} (optional, present if backfilled)
  
  agones.dev/sdk-should-allocate: {true|false}"
  
matchId - This is an ID unique to this match, not game. In this case, if it is a backfill,
the matchId of the original match will be used. The GameServer can use this to identify
which game the player should be put into (when using high density game servers).
```