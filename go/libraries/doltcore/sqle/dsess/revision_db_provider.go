// Copyright 2021 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dsess

import (
	"context"

	"github.com/dolthub/go-mysql-server/sql"
	"gopkg.in/src-d/go-errors.v1"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/filesys"
	"github.com/dolthub/dolt/go/store/types"
)

// ErrRevisionDbNotFound is thrown when a RevisionDatabaseProvider cannot find a specified revision database.
var ErrRevisionDbNotFound = errors.NewKind("revision database not found: '%s'")

// RevisionDatabaseProvider provides revision databases.
// In Dolt, commits and branches can be accessed as discrete databases
// using a Dolt-specific syntax: `my_database/my_branch`. Revision databases
// corresponding to historical commits in the repository will be read-only
// databases. Revision databases for branches will be read/write.
// TODO: most of the functionality in this interface has been exported to DoltDatabaseProvider, and what's left would better fit on RevisionDatabase
type RevisionDatabaseProvider interface {
	// GetRevisionForRevisionDatabase looks up the named database and returns the root database name as well as the
	// revision and any errors encountered. If the specified database is not a revision database, the root database
	// name will still be returned, and the revision will be an empty string.
	GetRevisionForRevisionDatabase(ctx *sql.Context, dbName string) (string, string, error)
}

// RevisionDatabase allows callers to query a revision database for the commit, branch, or tag it is pinned to. For
// example, when using a database with a branch revision specification, that database is only able to use that branch
// and cannot change branch heads. Calling `Revision` on that database will return the branch name. Similarly, for
// databases using a commit revision spec or a tag revision spec, Revision will return the commit or tag, respectively.
// Currently, only explicit branch names, commit hashes, and tag names are allowed as database revision specs. Other
// refspecs, such as "HEAD~2" are not supported yet.
type RevisionDatabase interface {
	// Revision returns the specific branch, commit, or tag to which this revision database has been pinned. Other
	// revision specifications (e.g. "HEAD~2") are not supported. If a database implements RevisionDatabase, but
	// is not pinned to a specific revision, the empty string is returned.
	Revision() string
}

// RemoteReadReplicaDatabase is a database that pulls from a connected remote when a transaction begins.
type RemoteReadReplicaDatabase interface {
	// ValidReplicaState returns whether this read replica is in a valid state to pull from the remote
	ValidReplicaState(ctx *sql.Context) bool
	// PullFromRemote performs a pull from the remote and returns any error encountered
	PullFromRemote(ctx *sql.Context) error
}

type DoltDatabaseProvider interface {
	sql.MutableDatabaseProvider
	RevisionDatabaseProvider
	// FileSystem returns the filesystem used by this provider, rooted at the data directory for all databases.
	FileSystem() filesys.Filesys
	// FileSystemForDatabase returns a filesystem, with the working directory set to the root directory
	// of the requested database. If the requested database isn't found, a database not found error
	// is returned.
	FileSystemForDatabase(dbname string) (filesys.Filesys, error)
	// GetRemoteDB returns the remote database for given env.Remote object using the local database's vrw, and
	// withCaching defines whether the remoteDB gets cached or not.
	// This function replaces env.Remote's GetRemoteDB method during SQL session to access dialer in order
	// to get remote database associated to the env.Remote object.
	GetRemoteDB(ctx context.Context, format *types.NomsBinFormat, r env.Remote, withCaching bool) (*doltdb.DoltDB, error)
	// CloneDatabaseFromRemote clones the database from the specified remoteURL as a new database in this provider.
	// dbName is the name for the new database, branch is an optional parameter indicating which branch to clone
	// (otherwise all branches are cloned), remoteName is the name for the remote created in the new database, and
	// remoteUrl is a URL (e.g. "file:///dbs/db1") or an <org>/<database> path indicating a database hosted on DoltHub.
	CloneDatabaseFromRemote(ctx *sql.Context, dbName, branch, remoteName, remoteUrl string, remoteParams map[string]string) error
	// SessionDatabase returns the SessionDatabase for the specified database, which may name a revision of a base
	// database.
	SessionDatabase(ctx *sql.Context, dbName string) (SessionDatabase, bool, error)
}