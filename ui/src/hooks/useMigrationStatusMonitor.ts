import { useEffect } from "react"
import { Migration, Phase } from "src/api/migrations/model"
import { useErrorHandler } from "./useErrorHandler"
import { useStatusTracker } from "./useStatusMonitor"


export const useMigrationStatusMonitor = (migrations: Migration[] = []) => {
  const { reportError } = useErrorHandler({
    component: "MigrationStatusMonitor",
  })
  const { statusTrackerRef, autoCleanup } = useStatusTracker<Phase>()

  useEffect(() => {
    if (!migrations || migrations.length === 0) return

    // Auto-cleanup old trackers  
    autoCleanup(migrations.map(m => m.metadata?.name))

    migrations.forEach((migration) => {
      const migrationName = migration.metadata?.name
      if (!migrationName) return

      const currentPhase = migration.status?.phase
      const tracker = statusTrackerRef.current[migrationName]

      // Initialize tracker for new migrations
      if (!tracker) {
        statusTrackerRef.current[migrationName] = {
          previousPhase: currentPhase,
        }
        return
      }

      // Skip if phase hasn't changed or already reported
      if (
        tracker.previousPhase === currentPhase ||
        tracker.lastReportedPhase === currentPhase
      ) {
        return
      }

      // Get error details from conditions
      const getErrorDetails = () => {
        const conditions = migration.status?.conditions || []
        // Find latest condition without sorting entire array
        const latestCondition = conditions.length > 0 
          ? conditions.reduce((latest, current) => {
              const currentTime = new Date(current.lastTransitionTime).getTime()
              const latestTime = new Date(latest.lastTransitionTime).getTime()
              return currentTime > latestTime ? current : latest
            })
          : null

        return {
          message: latestCondition?.message || `Migration ${currentPhase}`,
          reason: latestCondition?.reason || "Unknown",
          lastTransitionTime: latestCondition?.lastTransitionTime,
        }
      }

      // Handle migration execution failures
      if (
        currentPhase === Phase.Failed &&
        tracker.lastReportedPhase !== Phase.Failed
      ) {
        const errorDetails = getErrorDetails()


        // Report to Bugsnag
        const bugsnagError = new Error(
          `Migration execution failed: ${errorDetails.message}`
        )
        reportError(bugsnagError, {
          context: "migration-execution-failure",
          metadata: {
            migrationName,
            migrationPlan: migration.spec?.migrationPlan,
            vmName: migration.spec?.vmName,
            podRef: migration.spec?.podRef,
            previousPhase: tracker.previousPhase,
            currentPhase,
            errorMessage: errorDetails.message,
            errorReason: errorDetails.reason,
            failureTime: errorDetails.lastTransitionTime,
            namespace: migration.metadata?.namespace,
            conditions: migration.status?.conditions,
            action: "migration-execution-failed",
          },
        })

        console.error("Migration execution failed:", {
          migrationName,
          errorDetails,
          migration,
        })

        // Mark as reported
        statusTrackerRef.current[migrationName].lastReportedPhase = Phase.Failed
      }

      // Handle migration success (optional - for analytics)
      if (
        currentPhase === Phase.Succeeded &&
        tracker.lastReportedPhase !== Phase.Succeeded
      ) {

        // Mark as reported
        statusTrackerRef.current[migrationName].lastReportedPhase =
          Phase.Succeeded
      }

      // Update previous phase
      statusTrackerRef.current[migrationName].previousPhase = currentPhase
    })
  }, [migrations, reportError, autoCleanup])

  return {}
}
