# Introduction to Sugarkube

Welcome to this guide. This is intended for people who want to understand what problems Sugarkube solves and why it exists. A brief TLDR summary is: 

* Sugarkube is a release system that ships your applications and infrastructure code at the same time
* It doesn't require Kubernetes but provides extra features if you use it
* It works with any infrastructure you can script (cloud, local, on-prem, legacy) 
* It's compatible with any programming language
* It enables a multi-cloud strategy
* Sugarkube itself is a single golang binary that can be used for local development and be embedded in CI/CD pipelines. This simplifies local development by allowing developers to run almost exactly what the CI/CD tool will run.
* You can adopt it bit-by-bit while you test it out - you don't need to dedicate 3 months to migrate for an all-or-nothing release
* The project is currently in alpha
* Examples in these guides use Terraform and Helm, but both can be replaced by other similar tools as necessary (but things might be a bit tricky while we're still in alpha)

# The problem
Releasing software is difficult, especially when applications require certain infrastructure or cloud services, and their exact dependencies evolve over time. Coordinating infrastructure changes with code changes manually can work in the short term but be difficult as projects get more complex. Developers are typically under pressure to deliver products or prototypes rather than having several months to create a robust release pipeline. 

The outcome is that early decisions and processes that work with a very small team don't scale as the size of the team and complexity of the project increase. This can lead to a point in a project where the delivery rate drops to a crawl and developers end up fighting the release system, manually creating environments instead of just getting on with creating new features.

## Releases aren't that unique
We believe that there's a lot of commonality across release processes at different organisations. You may be tempted to think your stack is particularly unique, but unless you're operating at massive scale your release process can probably be summarised as:

> Release versions x, y and z of applications a, b and c through environments e, f and g, after possibly creating various bits of infrastructure in each environment first.

If that sounds like your organisation you're in luck. That's what Sugarkube aims to help you with!

# The ideal solution
We believe an ideal release pipeline should:

 * make it possible to spin up and tear down environments quickly and easily. This facilitates testing and makes your entire infrastructure more robust by preventing snowflakes (brittle environments with custom changes)
 * be able to easily reproduce the state of any cluster at an arbitrary point in time into a different cloud account
 * give you confidence in what you're releasing to prod by having tested that exact code in a lower environment (e.g. staging) 
 * scale to allow individual developers to work in ring-fenced environments - either on dedicated clusters or in isolated parts of larger development clusters.
 * let developers get to work quickly on tasks instead of having to waste large amounts of time setting up their clusters and cloud infrastructure first
 * allow developers to work locally as much as possible before developing in the Cloud 
 * not require you to use a particular CI/CD system during local development or testing in non-live environments (e.g. if your release pipeline is a custom Jenkins library you're forced to always deploy through Jenkins - this can complicate and slow down development)

# What is Sugarkube?
Sugarkube is a software release system that bundles your application code along with code to create any infrastructure it depends on, and versions it as a single unit. This means the releasable artefact is your application code + code to create dependent infrastructure. Because "app" is an overloaded term in software development, we call these bundles of
applications and infrastructure code "kapps" (originally from "Kubernetes app", but there's no requirement to use Sugarkube with Kubernetes any more).

Many other tools either only create infrastructure (Kubernetes, Terraform, CloudFormation, etc.) or release your applications (Helm, tarballs, whatever). This means you need some way of coordinating your application changes with the infrastructure they depend on, which can be complicated and error-prone.

Versioning code for your applications and infrastructure together is incredibly powerful. This idea means Sugarkube can:

* Recreate your clusters at any point in time. This makes it easy to create ephemeral clusters (e.g. a cluster per developer), and spin up/tear down testing/staging clusters.
* Support multi-cloud - A kapp can create one set of infrastructure when being installed into AWS and a different set when being installed into GCP/Azure or even on-premise or locally.
* Manage exactly which versions of these kapps (bundles) get released into each of your environments.
* Truly promote your applications and infrastructure through environments. An emphasise on portable artefacts (kapps) prevents you creating brittle snowflake environments.
* Install "slices" of your stack into different environments. For example if you have several monitoring and metric collection applications installed alongside your web sites, you can choose not to install the monitoring stack in your dev environment if you're not going to work on it.

All of the above make it simple to start working on new features without wasting time recreating infrastructure that your applications need. 

### Extra features for Kubernetes users
If you work with Kubernetes clusters Sugarkube provides additional features. It can launch clusters with several provisioners, e.g. Kops, Minikube (and more in future), and then configure them. For example it can patch Kops YAML configs before triggering an update to apply those changes. This makes it a useful tool for administering Kubernetes clusters. However its main benefit is that it allows you to create a cluster and install your applications (with dependent infrastructure) with a single command.

### Choose your own tools
Sugarkube doesn't force you to use any particular set of tools or technologies. It works with on-premise, legacy systems and infrastructure provided it's scriptable, and also with any programming language. You can adopt it bit-by-bit while you get used to it, and migrate more to it (or drop it) as you wish.

### Dealing with shared infrastructure 
One important thing to point out is that kapps must only create infrastructure that is only used by the application in the kapp. Any infrastructure that's shared between multiple applications/kapps must be created by another kapp (i.e. so you have one or several kapps dedicated to creating shared infrastructure like load balancers, hosted zone records, etc.).
These 'shared infrastructure' kapps therefore form the foundation for running certain groups of applications. So for example, you could create a shared infrastructure kapp to create your load balancer and hosted zone records, and configure it to be executed before executing kapps to install your web applications, etc.

# How it works 
## Kapps
The bundles of versioned application + infrastructure code are called "kapps". They're simply git repos where different directories contain a Makefile with some predefined targets. The git repos for kapps are tagged to create different versions.

If you decide to install your applications into a Kubernetes cluster using Helm chart and manage your infrastructure using Terraform code you can take advantage of our ready-made Makefiles that should cover 80-90% of use-cases. However, you have complete freedom to implement Makefiles as you want with several minor caveats. When Sugarkube runs it'll pass
several environment variables to the Makefile to allow it to modify its behaviour depending on which cloud provider is being targetted, the name of the target cluster, etc. 

**Note**: Although you don't have to use Kubernetes, Helm and Terraform with Sugarkube, we've made an assumption that you will while we're still in alpha. This allows us to simplify the problem-space and get something working in a more predictable setup instead of trying to please everyone immediately. So if you do choose not to use K8s, Helm and Terraform
you may find a few things don't work as expected. Please open an issue on Github to tell us about those scenarios if you run into them so we can track them. 

## Execution
When Sugarkube is executed, it:

1. Reads config files (which define your clusters, e.g. Kops on AWS or local Minikube, and the versions of which kapps to install into each cluster) 
1. Clones the relevant git repos containing your kapps at the specified version
1. Invokes `Make` on them passing various environment variables. The Makefiles tailor exactly what they do based on these environment variables. 

Most operations are run in parallel for speed (although that's configurable). This include cloning git repos and installing kapps.

The Makefile in each kapp acts as the interface between Sugarkube and exactly what a kapp does. Since our example kapps all use Helm + Terraform, we provide a set of default Makefiles that will:

* Lint Helm charts before installing them
* Initialise and execute Terraform code if a directory called `terraform_<provider>` exists

It's up to you to tailor each kapp's Makefile for your purposes. This approach makes kapps incredibly flexible and doesn't tie you into any particular programming language, tool (e.g. Helm/Terraform) or cloud provider. In future, we may even remove the dependency on Make, allowing you to invoke arbitrary scripts/binaries (e.g. if you'd rather write your releease scripts in your language of choice).

## Alpha software
To reiterate, Sugarkube is currently in alpha. To speed up development and to simplify the problem-space, examples for Sugarkube use Helm and Terraform. Helm is used because Kubernetes is becoming more and more popular so it makes sense to target it. Terraform is used because it will delete extraneous infrastructure, making it easy to tear down infrastructure. This is useful for testing and resetting environments. Despite this, it should be possible to use other tools as necessary but it might not be plain sailing for now. In future we hope to remove any dependencies on Helm, Terraform and Make.