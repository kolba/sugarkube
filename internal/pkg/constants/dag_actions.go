package constants

const DagActionInstall = "install"
const DagActionDelete = "delete"
const DagActionClean = "clean"
const DagActionTemplate = "template"
const DagActionOutput = "output"
const DagActionVars = "vars"
const ActionClusterUpdate = "cluster_update"
const ActionClusterDelete = "cluster_delete"
const ActionAddProviderVarsFiles = "add_provider_vars_files"
const ActionSkip = "skip" // The kapp will be not be installed or deleted, but everything else will happen. This is not
// the same as marking it as absent which may in future delete a kapp that's already installed but later marked as absent
