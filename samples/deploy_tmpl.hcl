# To check the full definition options, check
# https://hello.mlb.live-eu-1.meigas.cloud/services/
# https://www.nomadproject.io/docs/job-specification/index.html
job "{[.JOB_NAME]}{[.DEV_ENV]}" {
  # Where we will deploy
  region      = "{[.REGION]}"
  datacenters = ["{[.DATACENTER]}"]
{[- if .CRON_SCHEDULE]}
  type = "batch"
  periodic {
    cron             = "{[.CRON_SCHEDULE]}"
    prohibit_overlap = true
  }
{[- else]}
  # This is the default, so not really needed
  type = "service"
  # How to deploy when there's an update. One server every 30 seconds
  update {
    stagger      = "30s"
    max_parallel = 1
  }
{[- end]}

  # Namespace where we will deploy
  namespace = "{[.MEIGAS_NS]}"

  # Worker where to deploy
  constraint {
    attribute = "${node.class}"
    value     = "default"
  }

 meta{
{[- if and .POWERON .POWEROFF]}
      POWERON="{[.POWERON]}"
      POWEROFF="{[.POWEROFF]}"
{[- else ]}
      POWEROFF="false"
{[end]}
  }

  # Group is a set of tasks deployed on the same nomad client
  group "{[.GROUP_NAME]}" {
    count = "{[.INSTANCES]}"

{[- if not .DISABLE_CANARY]}
    # Canary deploy strategy
    update {
      max_parallel     = 1
      canary           = {[.INSTANCES]}
      min_healthy_time = "30s"
      healthy_deadline = "9m"
      auto_revert      = true
      auto_promote     = false
    }
{[end]}
    # We define that we want to use different hosts for each instance
    constraint {
      operator = "distinct_hosts"
      value    = "true"
    }

{[- if .CONNECT]}
    network {
          mbits = 10
          mode  = "bridge"
          port  "api" {
            to = {[.PORT_API]}
          }
        }
{[end]}

    # We define one task
    task "{[.COMMON_TASK_NAME]}" {
      # Driver to run the task
      driver = "docker"

      # Driver configuration
      config {
        image = "{[.DOCKER_REPOSITORY]}/{[.IMG_PROY]}/{[.IMG_NAME]}:{[.VERSION]}{[.SUBVERSION]}"
{[- if .DOCKER_FORCE_PULL]}
        force_pull = "true"
{[- end]}

        # Args is the actual command that's executed!
        args = [
{[- range $arg := .Filter "^ARG_ENV_\\d+$" ]}
          "{[$arg]}",
{[- end]}
        ]
        volumes = [
          # Use relative paths to rebind paths already in the allocation dir
          "config:/config"
        ]

        extra_hosts = [ {[.EXTRA_HOSTS]} ]

{[- if .PORT_API]}{[- if not .CONNECT]}
        port_map {
          api = {[.PORT_API]}
        }
{[- end]}{[- end]}
      }

      # Configuration for ECS Logs. Note: This might be deprecated
      meta {
        logger_namespace = "{[.SEMAAS_NS]}"
        logger_mrid = "{[.SEMAAS_MRID]}"
      }

      vault {
        policies = [{[.VAULT_POLICIES]}]
      }
{[if false]}
      # The container can use templates to store server configuration, environment vars
      # or whatever. On the template we have two ways of defining those.
{[- end]}
{[- if .TEMPLATE_DATA]}
      # Old template data, generated from a single variable
      {[.TEMPLATE_DATA]}
{[- end]}
{[- if false]}
  New way of generating templates, based on the CONSUL_KV_n, VAULT_SECRET_n and
  VAULT_ENVSECRET_n environment variables. Note that it should be easy to add more options
  here.

  Vault Templates will be populated from all the variables that have VAULT_SECRET_\d+ as a name
  pattern. The value of the variables is expected to be "Path;Key;Destination[;signal]". For example
  VAULT_SECRET_1="secret/this/is/a/path;whatever;secret/some/path.json"
  and
  VAULT_SECRET_2="secret/this/is/a/path;something;secret/some/path.json;1"

  The second sample means that when the value change a signal 1 will be sent to the process instead
  of restarting it.
{[end -]}
{[range $index, $vt := .Filter "^VAULT_SECRET_\\d+$" -]}
    {[$parts := $vt.Split ";" -]}
    {[$path := index $parts 0 -]}
    {[$key := index $parts 1 -]}
    {[$destination := index $parts 2 -]}
    {[$signal := "" -]}
    {[$changeMode := "restart" -]}
    {[if len $parts | lt 3 -]}
        {[$signal = index $parts 3 -]}
        {[$changeMode = "signal" -]}
    {[end]}
      template {
        data = "{{with secret \"{[$path]}\"}}{{print .Data.{[$key]}}}{{end}}"
        change_mode = "{[$changeMode]}"
        {[if eq $changeMode "signal" -]}
        change_signal = "{[$signal]}"
        {[end -]}
        destination = "{[$destination]}"
      }
{[end -]}

{[if false]}
  Consul Templates will be populated from all the variables that have `CONSUL_KV_\d+` as a name
  pattern. The value of the variables is expected to be `Path;Destination`. For example,
  `CONSUL_KV_1="config/whatever;config/config.json`
{[end -]}
{[range $ct := .Filter "^CONSUL_KV_\\d+$" -]}
  {[$parts := $ct.Split ";" -]}
  {[$path := index $parts 0 -]}
  {[$destination := index $parts 1]}
      template {
        data = "{{key \"{[$path]}\"}}"
        change_mode = "restart"
        destination = "{[$destination]}"
      }
 {[end -]}

{[if false]}
 Environment templates will be populated as follows:
  * The VAULT_ENV_FILE environment variable will be used as the destination file for the env config file
  * Vault Templates will be generated for all the variables that have `^VAULT_ENVSECRET_\d+$` as a
    name pattern. The value of the variables is expected to be as defined for Vault Templates above
    (that is "vault path;vault key;destination variable").
{[end -]}
{[if .VAULT_ENV_FILE]}
      template {
        data = <<EOH
        {[- range $vt := .Filter "^VAULT_ENVSECRET_\\d+$"]}
            {[$parts := $vt.Split ";" -]}
            {[$path := index $parts 0 -]}
            {[$key := index $parts 1 -]}
            {[$destination := index $parts 2 -]}
            {[$destination]}="{{with secret "{[$path]}"}}{{.Data.{[$key]}}}{{end}}"{[end]}
EOH
        destination = "{[.VAULT_ENV_FILE]}"
        env         = true
      }
{[- end]}

      env {
        APPLICATION_NAME = "{[.COMMON_TASK_NAME]}"
{[- range $elem := .Filter "^VAR_ENV_\\d+$" ]}
        {[$elemParts := $elem.Split ";" -]}
        {[$VarKey := index $elemParts 0 -]}
        {[$VarValue := index $elemParts 1 -]}
              {[$VarKey]} = "{[$VarValue]}",
{[- end]}
      }
      # Resources to reserve on the CPU, Memory and Network
      resources {
        cpu    = {[.CPU]}
        memory = {[.MEMORY]}

{[- if not .CRON_SCHEDULE]}{[- if not .CONNECT]}
        network {
          mbits = 10
          port  "api" {}
        }
{[- end]}{[- end]}
      }
{[- if .CONNECT]}
    }
{[- end]}

{[- if not .CRON_SCHEDULE]}

    # We can define the task as a consul-registered services, for which we can set healthchecks
    service {
      # NOMAD_TASK_NAME variable replaced because cannot be applied for consul connect services
      # https://www.nomadproject.io/docs/integrations/consul-connect
      name = "{[.COMMON_TASK_NAME]}"

      # To expose on the load balancer, use urlprefix
      tags = [
{[- if not .DISABLE_OLD_MLB]}{[if not .DISABLE_MLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.mlb.{[.DC_DOMAIN]}/",
{[end]}{[end]}
{[- if not .DISABLE_MLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.meigas.{[.DC_CTRL_DOMAIN]}/",
{[- end]}
{[- if not .DISABLE_IMLB]}
        "urlprefixint-{[.COMMON_TASK_NAME]}.imlb.{[.DC_DOMAIN]}/",
{[- end]}
{[- if not .DISABLE_OLD_SMLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.smlb.{[.DC_DOMAIN]}/",
{[- end]}
{[- if .DISABLE_AWS]}{[if not .DISABLE_SMLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.secaas.{[.DC_CTRL_DOMAIN]}/",
{[- end]}{[- end]}
{[- if not .DISABLE_AWS]}{[if not .DISABLE_SMLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.secaas.{[.AWS_DC_DOMAIN]}/",
{[- end]}{[- end]}
{[- if not .DISABLE_AWS]}{[if not .DISABLE_MLB]}
        "urlprefix-{[.COMMON_TASK_NAME]}.meigas.{[.AWS_DC_DOMAIN]}/",
{[- end]}{[- end]}
{[- if .CUSTOM_DOMAINS]}
  {[- range $elem := .CUSTOM_DOMAINS.Split " "]}
        "urlprefix-{[$elem]}/",
  {[- end]}
{[- end]}
      ]
      meta {
{[- if .USE_INGRESS]}
        ingress_match = "{[.COMMON_TASK_NAME]}/ {[- if .USE_CONNECT]} connect{[end]}"
    {[- if .INGRESS_MTLS]}
        ingress_verify_client_ca = "true"
      {[- if .INGRESS_MTLS_VERIFY_CERT]}
        ingress_allowed_dn_NAME = "{[.INGRESS_MTLS_VERIFY_CERT]}"
      {[- end]}
    {[- end]}
{[- end]}
      }
      # The port we will use for healthchecks
      port = "api"

      check {
        name     = "{[.COMMON_TASK_NAME]}-{[.CHECK_TYPE]}-check"
        type     = "{[.CHECK_TYPE]}"
        protocol = "{[.CHECK_PROTOCOL]}"
        path = "{[.CHECK_PATH]}"
        interval = "{[.CHECK_INTERVAL]}"
        timeout  = "2s"
        tls_skip_verify = true
      }
{[- if .EXTRA_CHECKS]}
      {[.EXTRA_CHECKS]}
{[- end]}
{[- if .CONFIGURED_HEALTHCHECKS]}{[$ctn := .COMMON_TASK_NAME]}
  {[range $hc := .CONFIGURED_HEALTHCHECKS.Fields -]}
    {[$hcParts := $hc.Split ":" -]}
    {[$hcFun := index $hcParts 1 -]}
    {[$hcPath := index $hcParts 0 -]}
    {[- if $hcFun]}
      check {
        name     = "{[$ctn]}-{[$hcFun]}-check"
        type     = "http"
        protocol = "https"
        path = "{[$hcPath]}"
        interval = "60s"
        timeout  = "2s"
        tls_skip_verify = true
      }
    {[end -]}
  {[- end]}
{[- end]}
    }
{[- end]}
{[- if not .CONNECT]}
  }
{[- end]}

{[- if .CONNECT]}

    service {
      name = "{[.JOB_NAME]}{[.DEV_ENV]}-sidecar"
      port = "{[.PORT_API]}"
      connect {
        sidecar_service {
          {[- if .SIDECAR_UPSTREAM_ENV_1]}
          tags = ["connect-proxy"] # Importat to override connect tags (if not they get inherited)
          proxy {
          {[- range $elem := .Filter "^SIDECAR_UPSTREAM_ENV_\\d+$" ]}
            {[$elemParts := $elem.Split ";" -]}
            {[$destinationName := index $elemParts 0 -]}
            {[$port := index $elemParts 1 -]}
            upstreams {
              destination_name = "{[$destinationName]}"
              local_bind_port  = {[$port]}
            }
          {[- end]}
          }
          {[- end]}
        }
      }
    }
{[- end]}
  }
}