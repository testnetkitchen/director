package version

var (
	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
	// Version is the application version
	Version = "0.0.1"
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
