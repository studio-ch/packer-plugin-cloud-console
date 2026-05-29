package builder

// Artifact describes the result of a studio-cp build. When push_image is set
// it points at the pushed OCI image; otherwise it identifies the (kept) VM.
type Artifact struct {
	VMName   string
	RegionID string

	// PushedImage is the OCI reference the VM was pushed to, if push_image
	// was configured. Empty otherwise.
	PushedImage string

	// PushedDigest is the manifest digest reported by the push job.
	PushedDigest string

	// RegisteredImageName is the tenant catalog name the pushed image was
	// registered as, if the push job registered it.
	RegisteredImageName string
}

func (*Artifact) BuilderId() string { return BuilderId }
func (*Artifact) Files() []string   { return nil }

func (a *Artifact) Id() string {
	if a.PushedImage != "" {
		if a.PushedDigest != "" {
			return a.PushedImage + "@" + a.PushedDigest
		}
		return a.PushedImage
	}
	return a.RegionID + "/" + a.VMName
}

func (a *Artifact) String() string { return "studio-cp VM: " + a.Id() }
func (*Artifact) State(string) any { return nil }
func (*Artifact) Destroy() error   { return nil }
