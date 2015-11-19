package dockerdevtools

func (v Version) DownloadURL() string {
	return v.downloadURL("Linux", "x86_64")
}
