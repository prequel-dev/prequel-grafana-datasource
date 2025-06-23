import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

// import { DataSourceInstanceSettings, ScopedVars, AnnotationEvent } from '@grafana/data';
// import { DataSourceWithBackend, getTemplateSrv, getBackendSrv } from '@grafana/runtime';

//import { AnnotationQueryEditor } from './components/AnnotationQueryEditor';
import { PrequelDataQuery, DataSourceOptions } from './types';
//import { firstValueFrom } from 'rxjs';

export class DataSource extends DataSourceWithBackend<PrequelDataQuery, DataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<DataSourceOptions>) {
    super(instanceSettings);
    this.annotations = {};
  }

  applyTemplateVariables(query: PrequelDataQuery, scopedVars: ScopedVars) {
    console.log('applyTemplateVariables', query, scopedVars);
    return {
      ...query,
      filter: getTemplateSrv().replace(query.filter, scopedVars),
    };
  }

  // filterQuery(query: MyQuery): boolean {
  //   // if no query has been provided, prevent the query from being executed
  //   return !!query.queryText;
  // }

  // async annotationQuery(options: any): Promise<AnnotationEvent[]> {
  //   console.log('annotationQuery', options);
  //   const { range, annotation } = options;
  //   const from = range.from.toISOString();
  //   const to = range.to.toISOString();

  //   const response = await firstValueFrom(getBackendSrv().fetch<AnnotationEvent[]>({
  //     method: 'POST',
  //     url: `/api/datasources/uid/${this.instanceSettings.uid}/resources/annotations`,
  //     headers: {
  //       'Content-Type': 'application/json',
  //     },
  //     data: {
  //       range: { from, to },
  //       annotation: {
  //         query: annotation.query,
  //       },
  //     },
  //   }));

  //   return response.data.map((item: any) => ({
  //     annotation,
  //     time: item.time,
  //     title: item.title || '',
  //     text: item.text || '',
  //     tags: item.tags || [],
  //   }));
  // }

}
