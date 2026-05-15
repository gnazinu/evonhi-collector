# Contexto: EvoNHI Collector (Go Agent)

## Estado Actual
El proyecto está inicializado siguiendo la arquitectura *Open Core*.
Actualmente es un MVP (Minimum Viable Product) en Go que logra conectarse a un clúster de Kubernetes usando el `kubeconfig` local y extrae exitosamente `ServiceAccounts` y `RoleBindings`, imprimiéndolos en texto plano en la consola. 

No tiene estructura de datos para exportación, no extrae la totalidad del RBAC, no se comunica con el SaaS y no está preparado para ejecutarse dentro del clúster (*in-cluster*).

## Roadmap Técnico para Grado Enterprise

Para que el Collector sea sólido, confiable para ventas B2B y compatible con `evonhi_core`, se deben completar las siguientes 4 fases en orden estricto:

### Fase 1: Estructuración y Extracción Completa (Mapeo a evonhi_core)
El motor en Python (`evonhi_core.models`) espera un grafo determinista. El Collector debe generar un JSON que coincida al 100% con esa expectativa.
- [ ] **Definir Structs en Go:** Crear los modelos con *tags* JSON para `ServiceAccount`, `Role`, `ClusterRole`, `RoleBinding`, `ClusterRoleBinding`, `Secret`, `Pod` y `Deployment`.
- [ ] **Completar la Extracción:** Ampliar el uso de `client-go` para listar los recursos faltantes.
- [ ] **Serialización:** Transformar la salida de consola en un único `ClusterStatePayload` en formato JSON.

### Fase 2: Ingestión HTTP (Conexión al SaaS)
El agente debe dejar de imprimir en consola y empezar a reportar a tu Control Plane.
- [ ] **Cliente HTTP:** Implementar una función `sender` nativa en Go para hacer un `POST` a `https://[TU-SAAS]/api/v1/connector-ingest/telemetry`.
- [ ] **Autenticación:** Añadir el Token del Conector (provisto por el SaaS al crear un entorno) en el header `Authorization: Bearer <token>`.
- [ ] **Resiliencia:** Implementar *retries* exponenciales y *timeouts* estrictos para no colgar el agente si el SaaS está caído.

### Fase 3: Seguridad y Confianza B2B (El Foso Comercial)
Nadie instalará el agente si creen que roba credenciales.
- [ ] **Anonimización:** Asegurar que el Collector NUNCA lea el campo `data` de los Kubernetes Secrets, solo la `metadata` (nombre, namespace, annotations).
- [ ] **Firma de Payload (Opcional pero recomendado):** Añadir una firma HMAC al JSON saliente usando un secreto compartido, para que el SaaS rechace payloads falsificados.

### Fase 4: Despliegue Nativo (In-Cluster)
El cliente no ejecutará un binario en su computadora; desplegará un contenedor en su clúster.
- [ ] **In-Cluster Config:** Modificar `main.go` para que, si no hay un flag `--kubeconfig`, use `rest.InClusterConfig()` por defecto.
- [ ] **Dockerfile:** Crear un contenedor distroless extremadamente ligero (ej. basado en `scratch` o `alpine`) compilando el binario estáticamente.
- [ ] **Manifiestos de Cliente (`deploy/manifests.yaml`):** Crear el archivo definitivo que el cliente aplicará:
  - Un `ServiceAccount` dedicado para el agente.
  - Un `ClusterRole` con permisos STRICTAMENTE de solo lectura (`get`, `list`) sobre recursos core y rbac.
  - Un `CronJob` o `Deployment` que corra el agente periódicamente.


