import { v4 as uuidv4 } from "uuid"
export const createStorageMappingJson = (params) => {
  const {
    name,
    namespace = "migration-system",
    storageMappings = [],
  } = params || {}
  return {
    apiVersion: "vjailbreak.k8s.stellaris-migrate.io/v1alpha1",
    kind: "StorageMapping",
    metadata: {
      name: name || uuidv4(),
      namespace: namespace,
    },
    spec: {
      storages: storageMappings,
    },
  }
}
