package cradle

import (
	"os"
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/fs"
)

/*
	Ensure the MVP filesystem, mkdir'ing and mutating as necessary.

	`ApplyDefaults` first (this refers to cwd).
*/
func MakeCradle(rootfsPath string, frm def.Formula) {
	ensureWorkingDir(rootfsPath, frm)
	ensureHomeDir(rootfsPath, frm.Action.Policy)
	ensureTempDir(rootfsPath)
	ensureIdentity(rootfsPath, frm.Action.Policy)
}

/*
	Ensure that the working directory specified in the formula exists, and
	if it had to be created, make it owned and writable by the user that
	the contained process will be launched as.  (If it already existed, do
	nothing; presumably you know what you're doing and intended whatever
	content is already there and whatever permissions are already in effect.)
*/
func ensureWorkingDir(rootfsPath string, frm def.Formula) {
	pth := filepath.Join(rootfsPath, frm.Action.Cwd)
	fs.MkdirAllWithAttribs(pth, fs.Metadata{
		Mode:       0755,
		ModTime:    def.Epochwhen,
		AccessTime: def.Epochwhen,
		Uid:        0, // TODO define map(policy)->posixUser, apply that here
		Gid:        0, // TODO define map(policy)->posixUser, apply that here
	})
}

func ensureHomeDir(rootfsPath string, policy def.Policy) {

}

/*
	Ensure `/tmp` exists and anyone can write there.
	The sticky bit will be applied and permissions set to 777.

	Edge case note: will follow symlinks.
*/
func ensureTempDir(rootfsPath string) {
	pth := filepath.Join(rootfsPath, "/tmp")
	stickyMode := os.FileMode(0777) | os.ModeSticky
	// try to chmod first
	err := os.Chmod(pth, stickyMode)
	if err == nil {
		return
	}
	// for unexpected complaints, bail
	if !os.IsNotExist(err) {
		panic(executor.SetupError.New("cradle: could not ensure reasonable /tmp: %s", err))
	}
	// mkdir if not exist
	if err := os.Mkdir(pth, stickyMode); err != nil {
		panic(executor.SetupError.New("cradle: could not ensure reasonable /tmp: %s", err))
	}
	// chmod it *again* because unit tests reveal that `os.Mkdir` is subject to umask
	if err := os.Chmod(pth, stickyMode); err != nil {
		panic(executor.SetupError.New("cradle: could not ensure reasonable /tmp: %s", err))
	}
}

/*
	Ensures the MVP filesystem considers configuration for the appropriate
	user account.

	Which is "the appropriate user account" varies according to the Policy.

	We define identity in terms that `nsswitch` would refer to as "compat":
	the ancient `/etc/{passwd,group,shadow}` files, because these are the
	most widely understood formats, and tend to keep working even in
	non-dynamically-linked programs (and different libc implementations,
	etc etc etc).

	We do not screw with a `/etc/nsswitch.conf` file if one exists, nor
	alter our behavior based on it -- cradle is about enforcing *absolute*
	minimum viable behaviors and fallbacks, not parsing and smoothing
	every concievable fractal of at-one-time-in-history-valid configuration.
*/
func ensureIdentity(rootfsPath string, policy def.Policy) {

}
