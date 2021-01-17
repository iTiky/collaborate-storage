# Collaborate storage

## Overview

### Data model

Server and client have a different data representation. This approach optimises server-client communication load and hides external meta data from user.

**Server**

```go
Item struct {
	Id        uuid.UUID
	Value     model.StorageValue
	IsDeleted bool
	UpdatedBy model.ClientId
	UpdatedAt time.Time
}
```

`model.StorageValue` is a type alias for `int32`.

Storage is using the "soft-delete" approach which makes possible to implement rollback, tracking changes and data versioning features.

Item has `UpdateBy` and `UpdatedAt` meta which might be helpful for collaborate storage use case.

Every object has a unique ID (UUID) which basically makes this storage a key-value storage. This approach has some pros and cons:

Pros:

* Insert / update operations optimization: easier to identify operation type (if key not exists -> insert);
* Delete operations optimization: easier to skip unnecessary storage IO if the specified key wasn't found;
* When there are concurrent insert / update operations performed by multiple clients, it makes easier to identify the target object;
* A sorted list of ojects is just an index and doesn't correlate with actual storage model;

Cons:

* For each object we add an additional 16 bytes which increases the memory usage by both server and client;
* Network load becomes significally higher (truth to be told, the ID is bigger than the data itself);

**Client**

```go
ListItem struct {
	Id    string
	Value StorageValue
}
```

From the clients perspective, data is a sorted list of items defined above.

### Document model

```go
DocumentHistory struct {
	sync.RWMutex
	// List of document versions
	documents []Document
	// Latest version storage state
	storage *Storage
	// The current document version
	latestVersion int
}

Document struct {
	Version int
	// Storage operations to apply on previous document version in order to upgrade it
	InputOperations []StorageOperation
	// Client model.StorageList operations to apply in order to upgrade it
	OutputOperations []model.ListOperation
}
```

The data is stored in a form of a Document. Document has a version and a number of operation which should be applied to the previous version in order to upgrade it.

Example: 

```
Document_v1 -> [ insert, update, update, delete ] operations -> Document_v2
```

That way we preserve all the data transformation history and we can (if needed) alter the Document history.

Example:

```
1. v0 -> v1 -> v2 -> v3
2. Alter v1 operations
3. v0 -> v1* -> v2* ->v3*
```

Also we can build data snapshots for every Document version and state.

For sure that approach vastly enlarges the disk and RAM used, but can implement something like Git's "squash": merge multiple Document versions into one, remove soft-deleted entries, etc.

### Transformation operations

`Document struct` introdused above has the following fields: `InputOperations` and `OutputOperations`.

`InputOperations` are `Storage` transformation blocks. `OutputOperations` are client's snapshot transformation operations.

Client snapshot transformation idea has the following points:

* Client downloads the latest snapshot containing all the data once;
* Client pulls diff operations to transform his local snapshot to the latest version;
* Network bandwidth duty is low;

Example:

```
1. Client 1: initial snapshot v0: [ 1, 3, 5, 7, 10 ]
2. Client 1: push insert operation (6)
3. Client 2: push delete operation (1)
4. Client 1: pulled the updates: [ {insert 6 at 3}, {delete 1 at 0} ]
5. Client 1: a new shapshot v1: [ 3, 5, 6, 7, 10 ]
```

If client misses an update (for example he has v3, but the server is at v5 now), the next updates poll would include v4 and v5 update operations.

### Offline mode

At the moment all the incoming update requests are pushed into the queue and are sorted by receive time (server time) before an actual apply. An offline user can push his changes as a batch operation. This approach leads to "who was the latest - wins". I'm not sure about this (may be the client time should be taken into consideration), the current implementation is "as I got it" one =)

### Server-client communication

Communication is done using the Golang RPC protocol as it was the fastets to implement.

The default port is `2412` (can be changed using command arguments).

At the moment the "push-pull" method is used. That way client has to poll the snapshot updates. The event-driven approach is a far better one, but it require a bit more development time.

## Source code

Code is divided into `cmd`, `model`, `service` and `storage`.

### cmd

Binary is build using [Cobra CLI library](https://github.com/spf13/cobra): server and client side are binded into one binary.

### model

* RPC request / response objects;
* Data snapshot objects used for client to work with (`model.StorageList`);

### storage

Keeps server side data representation and transformation functions for it.

* Object storage (`storage.Storage`, `storage.StorageOperation`) ;
* Document versioning storage (`storage.DocumentHistory`);

### service

* `/server` keeps the RPC server service with basic metrics collector;
* `/client` keeps the RPC client worker which generates load for the server and contains a basic metrics collector as well;

### build

* Docker and Docker compose manifests;
* `/resources` pregenerated mock storage files (I know it is a bad idea to store huge files in Git, but whatever }=) );

## Run

### Binary

As Docker has some RAM usage limitations, for the 10M storage size it might be easier to run server and clients using binary files.

1. Build

   ```bash
   make install
   cd ./build
   ```

   The binary would available at `${GOPATH}/src/github.com/itiky/collaborate-storage/build`.

2. Server start

   ```bash
   ./collaborate-storage server --file-path=./resources/doc_v0_10M.dat
   ```

   Command help with all available arguments can be obtained using:

   ```bash
   ./collaborate-storage server -h
   ```

3. Client start

   ```bash
   ./collaborate-storage client --client-id=1 --updates-max=15 --updates-period=1s --poll-period=500ms
   ```

   Command help with all available arguments can be obtained using:

   ```bash
   ./collaborate-storage client -h
   ```

   Multiple clients can be started in parallel. The `--client-id` argument is optional and only makes logs a bit more readable.

Document v0 state (initial snapshot) can be generated using:

```bash
./collaborate-storage generate --storage-size=10000000 --file-path="./doc_v0_10M.dat"
```

### Docker

By default Docker compose start a server with 4 clients using the 1M storage size.

1. Build

   ```bash
   make build-docker
   cd ./build
   ```

2. Start

   ```bash
   docker-compose up
   ```

### Monitor reports

Server and client print stat reports every 5 seconds.

**Server**

```
2021/01/18 00:25:15 Monitor:
2021/01/18 00:25:15   - Storate updates / s:   26.60
2021/01/18 00:25:15   - Diff requests / s:     4.00
2021/01/18 00:25:15   - Diff request dur [ms]: 26.24
```

* `Storate updates / s` - number of storage update request per second from clients;
* `Diff requests / s` - number of snapshot update requests per second from clients (diffs requests);
* `Diff request dur [ms]` - average response time for snapshot update requests in milliseconds;

**Client**

```
2021/01/18 00:26:30 Monitor:
2021/01/18 00:26:30   - Update requests / s:     7.60
2021/01/18 00:26:30   - Diff requests / s:       18.40
2021/01/18 00:26:30   - Update request dur [ms]: 0.29
2021/01/18 00:26:30   - Diff request dur [ms]:   1590.49
2021/01/18 00:26:30   - Consistancy dur [ms]:    1285.37
```

* `Update requests / s` - number of storage update request to server per second;
* `Diff requests / s` - number of snapshot update requests to server per second (polling);
* `Update request dur [ms]` - average response time for  storage update operations in milliseconds;
* `Diff request dur [ms]` - average response time for snapshot update request operations in milliseconds (polling);
* `Consistancy dur [ms]` - average time passed between client has send some updates and received them as a snapshot update in milliseconds (that includes snapshot transformation time);

The reports above are collected with the server running a 10M storage base and four clients pushing and pulling updates.

Client also prints the following logs:

```
2021/01/18 00:33:01 Client (3): [350.703µs] updates send: 13 ops
2021/01/18 00:33:02 Client (3): [619.286655ms] snapshot updated to v790: 16 ops (13 unhandled)
2021/01/18 00:33:02 Client (3): [491.616296ms] snapshot updated to v791: 13 ops (0 unhandled)
```

* 1st one: client has send 13 update operations and it took 350us;
* 2nd one: client has pulled a snapshot update to v790 with 16 snapshot transform operations (diffs) and it took 619ms;
* `13 unhandled` means client has 13 pushed operations that are not yet seen (replicated) within snapshot updates (v791 has them);

