# K8S-Event-Bot

A simple bot for your mattermost server, that can send you warning- and error-reports.

Features:
 * Send Warning on special event reasons
   * You can set the count-value in the config, when the bot will report the event and which event-reasons triggers an report.
   * You can submit a report and the bot would check this event again. The maintainers can see when the report was submitted.
 * Notify maintainers with direct messages on error-report **WIP**
