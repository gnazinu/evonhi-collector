{{/* Standard helpers */}}

{{- define "evo-collector.name" -}}
evo-collector
{{- end -}}

{{- define "evo-collector.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "evo-collector.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "evo-collector.labels" -}}
app.kubernetes.io/name: {{ include "evo-collector.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "evo-collector.tokenSecretName" -}}
{{- if .Values.token.create -}}
{{ include "evo-collector.fullname" . }}-token
{{- else -}}
{{ .Values.token.secretRef.name }}
{{- end -}}
{{- end -}}

{{- define "evo-collector.tokenSecretKey" -}}
{{- if .Values.token.create -}}
token
{{- else -}}
{{ .Values.token.secretRef.key | default "token" }}
{{- end -}}
{{- end -}}
