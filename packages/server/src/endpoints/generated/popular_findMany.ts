export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 8079,
          "end": 8087
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8088,
              "end": 8093
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8096,
                "end": 8101
              }
            },
            "loc": {
              "start": 8095,
              "end": 8101
            }
          },
          "loc": {
            "start": 8088,
            "end": 8101
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 8109,
                "end": 8114
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 8125,
                      "end": 8131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8125,
                    "end": 8131
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 8140,
                      "end": 8144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Api",
                            "loc": {
                              "start": 8166,
                              "end": 8169
                            }
                          },
                          "loc": {
                            "start": 8166,
                            "end": 8169
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Api_list",
                                "loc": {
                                  "start": 8191,
                                  "end": 8199
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8188,
                                "end": 8199
                              }
                            }
                          ],
                          "loc": {
                            "start": 8170,
                            "end": 8213
                          }
                        },
                        "loc": {
                          "start": 8159,
                          "end": 8213
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Note",
                            "loc": {
                              "start": 8233,
                              "end": 8237
                            }
                          },
                          "loc": {
                            "start": 8233,
                            "end": 8237
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Note_list",
                                "loc": {
                                  "start": 8259,
                                  "end": 8268
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8256,
                                "end": 8268
                              }
                            }
                          ],
                          "loc": {
                            "start": 8238,
                            "end": 8282
                          }
                        },
                        "loc": {
                          "start": 8226,
                          "end": 8282
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 8302,
                              "end": 8314
                            }
                          },
                          "loc": {
                            "start": 8302,
                            "end": 8314
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Organization_list",
                                "loc": {
                                  "start": 8336,
                                  "end": 8353
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8333,
                                "end": 8353
                              }
                            }
                          ],
                          "loc": {
                            "start": 8315,
                            "end": 8367
                          }
                        },
                        "loc": {
                          "start": 8295,
                          "end": 8367
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 8387,
                              "end": 8394
                            }
                          },
                          "loc": {
                            "start": 8387,
                            "end": 8394
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 8416,
                                  "end": 8428
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8413,
                                "end": 8428
                              }
                            }
                          ],
                          "loc": {
                            "start": 8395,
                            "end": 8442
                          }
                        },
                        "loc": {
                          "start": 8380,
                          "end": 8442
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Question",
                            "loc": {
                              "start": 8462,
                              "end": 8470
                            }
                          },
                          "loc": {
                            "start": 8462,
                            "end": 8470
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Question_list",
                                "loc": {
                                  "start": 8492,
                                  "end": 8505
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8489,
                                "end": 8505
                              }
                            }
                          ],
                          "loc": {
                            "start": 8471,
                            "end": 8519
                          }
                        },
                        "loc": {
                          "start": 8455,
                          "end": 8519
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 8539,
                              "end": 8546
                            }
                          },
                          "loc": {
                            "start": 8539,
                            "end": 8546
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 8568,
                                  "end": 8580
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8565,
                                "end": 8580
                              }
                            }
                          ],
                          "loc": {
                            "start": 8547,
                            "end": 8594
                          }
                        },
                        "loc": {
                          "start": 8532,
                          "end": 8594
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "SmartContract",
                            "loc": {
                              "start": 8614,
                              "end": 8627
                            }
                          },
                          "loc": {
                            "start": 8614,
                            "end": 8627
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "SmartContract_list",
                                "loc": {
                                  "start": 8649,
                                  "end": 8667
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8646,
                                "end": 8667
                              }
                            }
                          ],
                          "loc": {
                            "start": 8628,
                            "end": 8681
                          }
                        },
                        "loc": {
                          "start": 8607,
                          "end": 8681
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Standard",
                            "loc": {
                              "start": 8701,
                              "end": 8709
                            }
                          },
                          "loc": {
                            "start": 8701,
                            "end": 8709
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Standard_list",
                                "loc": {
                                  "start": 8731,
                                  "end": 8744
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8728,
                                "end": 8744
                              }
                            }
                          ],
                          "loc": {
                            "start": 8710,
                            "end": 8758
                          }
                        },
                        "loc": {
                          "start": 8694,
                          "end": 8758
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "User",
                            "loc": {
                              "start": 8778,
                              "end": 8782
                            }
                          },
                          "loc": {
                            "start": 8778,
                            "end": 8782
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "User_list",
                                "loc": {
                                  "start": 8804,
                                  "end": 8813
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8801,
                                "end": 8813
                              }
                            }
                          ],
                          "loc": {
                            "start": 8783,
                            "end": 8827
                          }
                        },
                        "loc": {
                          "start": 8771,
                          "end": 8827
                        }
                      }
                    ],
                    "loc": {
                      "start": 8145,
                      "end": 8837
                    }
                  },
                  "loc": {
                    "start": 8140,
                    "end": 8837
                  }
                }
              ],
              "loc": {
                "start": 8115,
                "end": 8843
              }
            },
            "loc": {
              "start": 8109,
              "end": 8843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8848,
                "end": 8856
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 8867,
                      "end": 8878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8867,
                    "end": 8878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8887,
                      "end": 8899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8887,
                    "end": 8899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8908,
                      "end": 8921
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8908,
                    "end": 8921
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 8930,
                      "end": 8951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8930,
                    "end": 8951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8960,
                      "end": 8976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8960,
                    "end": 8976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8985,
                      "end": 9002
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8985,
                    "end": 9002
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 9011,
                      "end": 9027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9011,
                    "end": 9027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorSmartContract",
                    "loc": {
                      "start": 9036,
                      "end": 9058
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9036,
                    "end": 9058
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 9067,
                      "end": 9084
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9067,
                    "end": 9084
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 9093,
                      "end": 9106
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9093,
                    "end": 9106
                  }
                }
              ],
              "loc": {
                "start": 8857,
                "end": 9112
              }
            },
            "loc": {
              "start": 8848,
              "end": 9112
            }
          }
        ],
        "loc": {
          "start": 8103,
          "end": 9116
        }
      },
      "loc": {
        "start": 8079,
        "end": 9116
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 947,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 950,
          "end": 960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 961,
          "end": 971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 961,
        "end": 971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 972,
          "end": 977
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 972,
        "end": 977
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 978,
          "end": 983
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 978,
        "end": 983
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 984,
          "end": 989
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1003,
                  "end": 1015
                }
              },
              "loc": {
                "start": 1003,
                "end": 1015
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1029,
                      "end": 1045
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1026,
                    "end": 1045
                  }
                }
              ],
              "loc": {
                "start": 1016,
                "end": 1051
              }
            },
            "loc": {
              "start": 996,
              "end": 1051
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1063,
                  "end": 1067
                }
              },
              "loc": {
                "start": 1063,
                "end": 1067
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1081,
                      "end": 1089
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1078,
                    "end": 1089
                  }
                }
              ],
              "loc": {
                "start": 1068,
                "end": 1095
              }
            },
            "loc": {
              "start": 1056,
              "end": 1095
            }
          }
        ],
        "loc": {
          "start": 990,
          "end": 1097
        }
      },
      "loc": {
        "start": 984,
        "end": 1097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1098,
          "end": 1101
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1108,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1122,
                "end": 1131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1131
            }
          }
        ],
        "loc": {
          "start": 1102,
          "end": 1133
        }
      },
      "loc": {
        "start": 1098,
        "end": 1133
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1165,
          "end": 1173
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1180,
                "end": 1192
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1203,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1214,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1214,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1231,
                      "end": 1242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1251,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1251,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 1264,
                      "end": 1269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1284,
                            "end": 1286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1284,
                          "end": 1286
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 1299,
                            "end": 1308
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1299,
                          "end": 1308
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1321,
                            "end": 1325
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1321,
                          "end": 1325
                        }
                      }
                    ],
                    "loc": {
                      "start": 1270,
                      "end": 1335
                    }
                  },
                  "loc": {
                    "start": 1264,
                    "end": 1335
                  }
                }
              ],
              "loc": {
                "start": 1193,
                "end": 1341
              }
            },
            "loc": {
              "start": 1180,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1346,
                "end": 1348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1353,
                "end": 1363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1353,
              "end": 1363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1368,
                "end": 1378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1368,
              "end": 1378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1383,
                "end": 1391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1383,
              "end": 1391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1396,
                "end": 1405
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1396,
              "end": 1405
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1410,
                "end": 1422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1410,
              "end": 1422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1427,
                "end": 1439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1427,
              "end": 1439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1444,
                "end": 1456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1444,
              "end": 1456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1461,
                "end": 1464
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 1475,
                      "end": 1485
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1475,
                    "end": 1485
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1494,
                      "end": 1501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1494,
                    "end": 1501
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1510,
                      "end": 1519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1528,
                      "end": 1537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1528,
                    "end": 1537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1546,
                      "end": 1555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1546,
                    "end": 1555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1564,
                      "end": 1570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1579,
                      "end": 1586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1579,
                    "end": 1586
                  }
                }
              ],
              "loc": {
                "start": 1465,
                "end": 1592
              }
            },
            "loc": {
              "start": 1461,
              "end": 1592
            }
          }
        ],
        "loc": {
          "start": 1174,
          "end": 1594
        }
      },
      "loc": {
        "start": 1165,
        "end": 1594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1595,
          "end": 1597
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1595,
        "end": 1597
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1598,
          "end": 1608
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1598,
        "end": 1608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1609,
          "end": 1619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1609,
        "end": 1619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1620,
          "end": 1629
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1620,
        "end": 1629
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1630,
          "end": 1641
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1630,
        "end": 1641
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1642,
          "end": 1648
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1658,
                "end": 1668
              }
            },
            "directives": [],
            "loc": {
              "start": 1655,
              "end": 1668
            }
          }
        ],
        "loc": {
          "start": 1649,
          "end": 1670
        }
      },
      "loc": {
        "start": 1642,
        "end": 1670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1671,
          "end": 1676
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1690,
                  "end": 1702
                }
              },
              "loc": {
                "start": 1690,
                "end": 1702
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1716,
                      "end": 1732
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1713,
                    "end": 1732
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 1738
              }
            },
            "loc": {
              "start": 1683,
              "end": 1738
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1750,
                  "end": 1754
                }
              },
              "loc": {
                "start": 1750,
                "end": 1754
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1768,
                      "end": 1776
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1765,
                    "end": 1776
                  }
                }
              ],
              "loc": {
                "start": 1755,
                "end": 1782
              }
            },
            "loc": {
              "start": 1743,
              "end": 1782
            }
          }
        ],
        "loc": {
          "start": 1677,
          "end": 1784
        }
      },
      "loc": {
        "start": 1671,
        "end": 1784
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1785,
          "end": 1796
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1785,
        "end": 1796
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1797,
          "end": 1811
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1797,
        "end": 1811
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1812,
          "end": 1817
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1812,
        "end": 1817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1818,
          "end": 1827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1818,
        "end": 1827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1828,
          "end": 1832
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1842,
                "end": 1850
              }
            },
            "directives": [],
            "loc": {
              "start": 1839,
              "end": 1850
            }
          }
        ],
        "loc": {
          "start": 1833,
          "end": 1852
        }
      },
      "loc": {
        "start": 1828,
        "end": 1852
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1853,
          "end": 1867
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1853,
        "end": 1867
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1868,
          "end": 1873
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1868,
        "end": 1873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1874,
          "end": 1877
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1884,
                "end": 1893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1884,
              "end": 1893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1898,
                "end": 1909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1898,
              "end": 1909
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1914,
                "end": 1925
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1914,
              "end": 1925
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1930,
                "end": 1939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1930,
              "end": 1939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1944,
                "end": 1951
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1944,
              "end": 1951
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1956,
                "end": 1964
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1956,
              "end": 1964
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1969,
                "end": 1981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1969,
              "end": 1981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1986,
                "end": 1994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1986,
              "end": 1994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1999,
                "end": 2007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1999,
              "end": 2007
            }
          }
        ],
        "loc": {
          "start": 1878,
          "end": 2009
        }
      },
      "loc": {
        "start": 1874,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2040,
          "end": 2042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2040,
        "end": 2042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2043,
          "end": 2052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2043,
        "end": 2052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2100,
          "end": 2102
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2100,
        "end": 2102
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2103,
          "end": 2114
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2103,
        "end": 2114
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2115,
          "end": 2121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2115,
        "end": 2121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2122,
          "end": 2132
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2122,
        "end": 2132
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2133,
          "end": 2143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2133,
        "end": 2143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2144,
          "end": 2162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2144,
        "end": 2162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2163,
          "end": 2172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2163,
        "end": 2172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2173,
          "end": 2186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2173,
        "end": 2186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2187,
          "end": 2199
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2187,
        "end": 2199
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2200,
          "end": 2212
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2200,
        "end": 2212
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2213,
          "end": 2225
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2213,
        "end": 2225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2226,
          "end": 2235
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2226,
        "end": 2235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2236,
          "end": 2240
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2250,
                "end": 2258
              }
            },
            "directives": [],
            "loc": {
              "start": 2247,
              "end": 2258
            }
          }
        ],
        "loc": {
          "start": 2241,
          "end": 2260
        }
      },
      "loc": {
        "start": 2236,
        "end": 2260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2261,
          "end": 2273
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2280,
                "end": 2282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2280,
              "end": 2282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2287,
                "end": 2295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2287,
              "end": 2295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2300,
                "end": 2303
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2300,
              "end": 2303
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2308,
                "end": 2312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2308,
              "end": 2312
            }
          }
        ],
        "loc": {
          "start": 2274,
          "end": 2314
        }
      },
      "loc": {
        "start": 2261,
        "end": 2314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2315,
          "end": 2318
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2325,
                "end": 2338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2325,
              "end": 2338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2343,
                "end": 2352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2343,
              "end": 2352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2357,
                "end": 2368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2357,
              "end": 2368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2373,
                "end": 2382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2373,
              "end": 2382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2387,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2387,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2401,
                "end": 2408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2401,
              "end": 2408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2413,
                "end": 2425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2413,
              "end": 2425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2430,
                "end": 2438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2430,
              "end": 2438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2443,
                "end": 2457
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2468,
                      "end": 2470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2468,
                    "end": 2470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2479,
                      "end": 2489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2479,
                    "end": 2489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2498,
                      "end": 2508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2498,
                    "end": 2508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2517,
                      "end": 2524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2517,
                    "end": 2524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2533,
                      "end": 2544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2533,
                    "end": 2544
                  }
                }
              ],
              "loc": {
                "start": 2458,
                "end": 2550
              }
            },
            "loc": {
              "start": 2443,
              "end": 2550
            }
          }
        ],
        "loc": {
          "start": 2319,
          "end": 2552
        }
      },
      "loc": {
        "start": 2315,
        "end": 2552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2599,
          "end": 2601
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2599,
        "end": 2601
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2602,
          "end": 2613
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2602,
        "end": 2613
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2614,
          "end": 2620
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2614,
        "end": 2620
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2621,
          "end": 2633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2621,
        "end": 2633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2634,
          "end": 2637
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2644,
                "end": 2657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2644,
              "end": 2657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2662,
                "end": 2671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2662,
              "end": 2671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2676,
                "end": 2687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2676,
              "end": 2687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2692,
                "end": 2701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2692,
              "end": 2701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2706,
                "end": 2715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2706,
              "end": 2715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2720,
                "end": 2727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2720,
              "end": 2727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2732,
                "end": 2744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2732,
              "end": 2744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2749,
                "end": 2757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2749,
              "end": 2757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2762,
                "end": 2776
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2787,
                      "end": 2789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2787,
                    "end": 2789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2798,
                      "end": 2808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2798,
                    "end": 2808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2817,
                      "end": 2827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2817,
                    "end": 2827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2836,
                      "end": 2843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2836,
                    "end": 2843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2852,
                      "end": 2863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2852,
                    "end": 2863
                  }
                }
              ],
              "loc": {
                "start": 2777,
                "end": 2869
              }
            },
            "loc": {
              "start": 2762,
              "end": 2869
            }
          }
        ],
        "loc": {
          "start": 2638,
          "end": 2871
        }
      },
      "loc": {
        "start": 2634,
        "end": 2871
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2909,
          "end": 2917
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2924,
                "end": 2936
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2947,
                      "end": 2949
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2947,
                    "end": 2949
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2958,
                      "end": 2966
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2958,
                    "end": 2966
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2975,
                      "end": 2986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2975,
                    "end": 2986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2995,
                      "end": 2999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2995,
                    "end": 2999
                  }
                }
              ],
              "loc": {
                "start": 2937,
                "end": 3005
              }
            },
            "loc": {
              "start": 2924,
              "end": 3005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3010,
                "end": 3012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3010,
              "end": 3012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3017,
                "end": 3027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3017,
              "end": 3027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3032,
                "end": 3042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3032,
              "end": 3042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3047,
                "end": 3063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3047,
              "end": 3063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3068,
                "end": 3076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3068,
              "end": 3076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3081,
                "end": 3090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3081,
              "end": 3090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3095,
                "end": 3107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3095,
              "end": 3107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3112,
                "end": 3128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3112,
              "end": 3128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3133,
                "end": 3143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3133,
              "end": 3143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3148,
                "end": 3160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3148,
              "end": 3160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3165,
                "end": 3177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3165,
              "end": 3177
            }
          }
        ],
        "loc": {
          "start": 2918,
          "end": 3179
        }
      },
      "loc": {
        "start": 2909,
        "end": 3179
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3180,
          "end": 3182
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3180,
        "end": 3182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3183,
          "end": 3193
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3183,
        "end": 3193
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3194,
          "end": 3204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3194,
        "end": 3204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3205,
          "end": 3214
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3205,
        "end": 3214
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3215,
          "end": 3226
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3215,
        "end": 3226
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3227,
          "end": 3233
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 3243,
                "end": 3253
              }
            },
            "directives": [],
            "loc": {
              "start": 3240,
              "end": 3253
            }
          }
        ],
        "loc": {
          "start": 3234,
          "end": 3255
        }
      },
      "loc": {
        "start": 3227,
        "end": 3255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3256,
          "end": 3261
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 3275,
                  "end": 3287
                }
              },
              "loc": {
                "start": 3275,
                "end": 3287
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 3301,
                      "end": 3317
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3298,
                    "end": 3317
                  }
                }
              ],
              "loc": {
                "start": 3288,
                "end": 3323
              }
            },
            "loc": {
              "start": 3268,
              "end": 3323
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3335,
                  "end": 3339
                }
              },
              "loc": {
                "start": 3335,
                "end": 3339
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3353,
                      "end": 3361
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3350,
                    "end": 3361
                  }
                }
              ],
              "loc": {
                "start": 3340,
                "end": 3367
              }
            },
            "loc": {
              "start": 3328,
              "end": 3367
            }
          }
        ],
        "loc": {
          "start": 3262,
          "end": 3369
        }
      },
      "loc": {
        "start": 3256,
        "end": 3369
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3370,
          "end": 3381
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3370,
        "end": 3381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3382,
          "end": 3396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3382,
        "end": 3396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3397,
          "end": 3402
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3397,
        "end": 3402
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3403,
          "end": 3412
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3403,
        "end": 3412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3413,
          "end": 3417
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 3427,
                "end": 3435
              }
            },
            "directives": [],
            "loc": {
              "start": 3424,
              "end": 3435
            }
          }
        ],
        "loc": {
          "start": 3418,
          "end": 3437
        }
      },
      "loc": {
        "start": 3413,
        "end": 3437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3438,
          "end": 3452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3438,
        "end": 3452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3453,
          "end": 3458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3453,
        "end": 3458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3459,
          "end": 3462
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 3469,
                "end": 3478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3469,
              "end": 3478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3483,
                "end": 3494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3483,
              "end": 3494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3499,
                "end": 3510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3499,
              "end": 3510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3515,
                "end": 3524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3515,
              "end": 3524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3529,
                "end": 3536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3529,
              "end": 3536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3541,
                "end": 3549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3541,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3554,
                "end": 3566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3554,
              "end": 3566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3571,
                "end": 3579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3571,
              "end": 3579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3584,
                "end": 3592
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3584,
              "end": 3592
            }
          }
        ],
        "loc": {
          "start": 3463,
          "end": 3594
        }
      },
      "loc": {
        "start": 3459,
        "end": 3594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3631,
          "end": 3633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3631,
        "end": 3633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3634,
          "end": 3643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3634,
        "end": 3643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3683,
          "end": 3695
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3702,
                "end": 3704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3702,
              "end": 3704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3709,
                "end": 3717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3709,
              "end": 3717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3722,
                "end": 3733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3722,
              "end": 3733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3738,
                "end": 3742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3738,
              "end": 3742
            }
          }
        ],
        "loc": {
          "start": 3696,
          "end": 3744
        }
      },
      "loc": {
        "start": 3683,
        "end": 3744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3745,
          "end": 3747
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3745,
        "end": 3747
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3748,
          "end": 3758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3748,
        "end": 3758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3759,
          "end": 3769
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3759,
        "end": 3769
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3770,
          "end": 3779
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3786,
                "end": 3788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3786,
              "end": 3788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3793,
                "end": 3803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3793,
              "end": 3803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3808,
                "end": 3818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3808,
              "end": 3818
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3823,
                "end": 3834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3823,
              "end": 3834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3839,
                "end": 3845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3839,
              "end": 3845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3850,
                "end": 3855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3850,
              "end": 3855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3860,
                "end": 3880
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3860,
              "end": 3880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3885,
                "end": 3889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3885,
              "end": 3889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3894,
                "end": 3906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3894,
              "end": 3906
            }
          }
        ],
        "loc": {
          "start": 3780,
          "end": 3908
        }
      },
      "loc": {
        "start": 3770,
        "end": 3908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3909,
          "end": 3926
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3909,
        "end": 3926
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3927,
          "end": 3936
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3927,
        "end": 3936
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3937,
          "end": 3942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3937,
        "end": 3942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3943,
          "end": 3952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3943,
        "end": 3952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 3953,
          "end": 3965
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3953,
        "end": 3965
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 3966,
          "end": 3979
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3966,
        "end": 3979
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 3980,
          "end": 3992
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3980,
        "end": 3992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 3993,
          "end": 4002
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 4016,
                  "end": 4019
                }
              },
              "loc": {
                "start": 4016,
                "end": 4019
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 4033,
                      "end": 4040
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4030,
                    "end": 4040
                  }
                }
              ],
              "loc": {
                "start": 4020,
                "end": 4046
              }
            },
            "loc": {
              "start": 4009,
              "end": 4046
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4058,
                  "end": 4062
                }
              },
              "loc": {
                "start": 4058,
                "end": 4062
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4076,
                      "end": 4084
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4073,
                    "end": 4084
                  }
                }
              ],
              "loc": {
                "start": 4063,
                "end": 4090
              }
            },
            "loc": {
              "start": 4051,
              "end": 4090
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 4102,
                  "end": 4114
                }
              },
              "loc": {
                "start": 4102,
                "end": 4114
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 4128,
                      "end": 4144
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4125,
                    "end": 4144
                  }
                }
              ],
              "loc": {
                "start": 4115,
                "end": 4150
              }
            },
            "loc": {
              "start": 4095,
              "end": 4150
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4162,
                  "end": 4169
                }
              },
              "loc": {
                "start": 4162,
                "end": 4169
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4183,
                      "end": 4194
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4180,
                    "end": 4194
                  }
                }
              ],
              "loc": {
                "start": 4170,
                "end": 4200
              }
            },
            "loc": {
              "start": 4155,
              "end": 4200
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4212,
                  "end": 4219
                }
              },
              "loc": {
                "start": 4212,
                "end": 4219
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4233,
                      "end": 4244
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4230,
                    "end": 4244
                  }
                }
              ],
              "loc": {
                "start": 4220,
                "end": 4250
              }
            },
            "loc": {
              "start": 4205,
              "end": 4250
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "SmartContract",
                "loc": {
                  "start": 4262,
                  "end": 4275
                }
              },
              "loc": {
                "start": 4262,
                "end": 4275
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "SmartContract_nav",
                    "loc": {
                      "start": 4289,
                      "end": 4306
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4286,
                    "end": 4306
                  }
                }
              ],
              "loc": {
                "start": 4276,
                "end": 4312
              }
            },
            "loc": {
              "start": 4255,
              "end": 4312
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4324,
                  "end": 4332
                }
              },
              "loc": {
                "start": 4324,
                "end": 4332
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4346,
                      "end": 4358
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4343,
                    "end": 4358
                  }
                }
              ],
              "loc": {
                "start": 4333,
                "end": 4364
              }
            },
            "loc": {
              "start": 4317,
              "end": 4364
            }
          }
        ],
        "loc": {
          "start": 4003,
          "end": 4366
        }
      },
      "loc": {
        "start": 3993,
        "end": 4366
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4367,
          "end": 4371
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 4381,
                "end": 4389
              }
            },
            "directives": [],
            "loc": {
              "start": 4378,
              "end": 4389
            }
          }
        ],
        "loc": {
          "start": 4372,
          "end": 4391
        }
      },
      "loc": {
        "start": 4367,
        "end": 4391
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4392,
          "end": 4395
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4402,
                "end": 4410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4402,
              "end": 4410
            }
          }
        ],
        "loc": {
          "start": 4396,
          "end": 4412
        }
      },
      "loc": {
        "start": 4392,
        "end": 4412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4450,
          "end": 4458
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4465,
                "end": 4477
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4488,
                      "end": 4490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4488,
                    "end": 4490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4499,
                      "end": 4507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4499,
                    "end": 4507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4516,
                      "end": 4527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4516,
                    "end": 4527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4536,
                      "end": 4548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4536,
                    "end": 4548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4557,
                      "end": 4561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4557,
                    "end": 4561
                  }
                }
              ],
              "loc": {
                "start": 4478,
                "end": 4567
              }
            },
            "loc": {
              "start": 4465,
              "end": 4567
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4572,
                "end": 4574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4572,
              "end": 4574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4579,
                "end": 4589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4579,
              "end": 4589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4594,
                "end": 4604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4594,
              "end": 4604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4609,
                "end": 4620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4609,
              "end": 4620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4625,
                "end": 4638
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4625,
              "end": 4638
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4643,
                "end": 4653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4643,
              "end": 4653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4658,
                "end": 4667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4658,
              "end": 4667
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4672,
                "end": 4680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4672,
              "end": 4680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4685,
                "end": 4694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4685,
              "end": 4694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4699,
                "end": 4709
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4699,
              "end": 4709
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4714,
                "end": 4726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4714,
              "end": 4726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4731,
                "end": 4745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4731,
              "end": 4745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4750,
                "end": 4771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4750,
              "end": 4771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 4776,
                "end": 4787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4776,
              "end": 4787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4792,
                "end": 4804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4792,
              "end": 4804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4809,
                "end": 4821
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4809,
              "end": 4821
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4826,
                "end": 4839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4826,
              "end": 4839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4844,
                "end": 4866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4844,
              "end": 4866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4871,
                "end": 4881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4871,
              "end": 4881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4886,
                "end": 4897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4886,
              "end": 4897
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4902,
                "end": 4912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4902,
              "end": 4912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4917,
                "end": 4931
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4917,
              "end": 4931
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4936,
                "end": 4948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4936,
              "end": 4948
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4953,
                "end": 4965
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4953,
              "end": 4965
            }
          }
        ],
        "loc": {
          "start": 4459,
          "end": 4967
        }
      },
      "loc": {
        "start": 4450,
        "end": 4967
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4968,
          "end": 4970
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4968,
        "end": 4970
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4971,
          "end": 4981
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4971,
        "end": 4981
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4982,
          "end": 4992
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4982,
        "end": 4992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4993,
          "end": 5003
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4993,
        "end": 5003
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5004,
          "end": 5013
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5004,
        "end": 5013
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5014,
          "end": 5025
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5014,
        "end": 5025
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5026,
          "end": 5032
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 5042,
                "end": 5052
              }
            },
            "directives": [],
            "loc": {
              "start": 5039,
              "end": 5052
            }
          }
        ],
        "loc": {
          "start": 5033,
          "end": 5054
        }
      },
      "loc": {
        "start": 5026,
        "end": 5054
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5055,
          "end": 5060
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 5074,
                  "end": 5086
                }
              },
              "loc": {
                "start": 5074,
                "end": 5086
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5100,
                      "end": 5116
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5097,
                    "end": 5116
                  }
                }
              ],
              "loc": {
                "start": 5087,
                "end": 5122
              }
            },
            "loc": {
              "start": 5067,
              "end": 5122
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5134,
                  "end": 5138
                }
              },
              "loc": {
                "start": 5134,
                "end": 5138
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5152,
                      "end": 5160
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5149,
                    "end": 5160
                  }
                }
              ],
              "loc": {
                "start": 5139,
                "end": 5166
              }
            },
            "loc": {
              "start": 5127,
              "end": 5166
            }
          }
        ],
        "loc": {
          "start": 5061,
          "end": 5168
        }
      },
      "loc": {
        "start": 5055,
        "end": 5168
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5169,
          "end": 5180
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5169,
        "end": 5180
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5181,
          "end": 5195
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5181,
        "end": 5195
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5196,
          "end": 5201
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5196,
        "end": 5201
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5202,
          "end": 5211
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5202,
        "end": 5211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5212,
          "end": 5216
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 5226,
                "end": 5234
              }
            },
            "directives": [],
            "loc": {
              "start": 5223,
              "end": 5234
            }
          }
        ],
        "loc": {
          "start": 5217,
          "end": 5236
        }
      },
      "loc": {
        "start": 5212,
        "end": 5236
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5237,
          "end": 5251
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5237,
        "end": 5251
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5252,
          "end": 5257
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5252,
        "end": 5257
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5258,
          "end": 5261
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canComment",
              "loc": {
                "start": 5268,
                "end": 5278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5268,
              "end": 5278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5283,
                "end": 5292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5283,
              "end": 5292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5297,
                "end": 5308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5297,
              "end": 5308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5313,
                "end": 5322
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5313,
              "end": 5322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5327,
                "end": 5334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5327,
              "end": 5334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5339,
                "end": 5347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5339,
              "end": 5347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5352,
                "end": 5364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5352,
              "end": 5364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5369,
                "end": 5377
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5369,
              "end": 5377
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5382,
                "end": 5390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5382,
              "end": 5390
            }
          }
        ],
        "loc": {
          "start": 5262,
          "end": 5392
        }
      },
      "loc": {
        "start": 5258,
        "end": 5392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5429,
          "end": 5431
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5429,
        "end": 5431
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5432,
          "end": 5442
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5432,
        "end": 5442
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5443,
          "end": 5452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5443,
        "end": 5452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5502,
          "end": 5510
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5517,
                "end": 5529
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5540,
                      "end": 5542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5540,
                    "end": 5542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5551,
                      "end": 5559
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5551,
                    "end": 5559
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5568,
                      "end": 5579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5568,
                    "end": 5579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5588,
                      "end": 5600
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5588,
                    "end": 5600
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5609,
                      "end": 5613
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5609,
                    "end": 5613
                  }
                }
              ],
              "loc": {
                "start": 5530,
                "end": 5619
              }
            },
            "loc": {
              "start": 5517,
              "end": 5619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5624,
                "end": 5626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5624,
              "end": 5626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5631,
                "end": 5641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5631,
              "end": 5641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5646,
                "end": 5656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5646,
              "end": 5656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5661,
                "end": 5671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5661,
              "end": 5671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 5676,
                "end": 5685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5676,
              "end": 5685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5690,
                "end": 5698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5690,
              "end": 5698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5703,
                "end": 5712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5703,
              "end": 5712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5717,
                "end": 5724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5717,
              "end": 5724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 5729,
                "end": 5741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5729,
              "end": 5741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 5746,
                "end": 5753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5746,
              "end": 5753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5758,
                "end": 5770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5758,
              "end": 5770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5775,
                "end": 5787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5775,
              "end": 5787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5792,
                "end": 5805
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5792,
              "end": 5805
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5810,
                "end": 5832
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5810,
              "end": 5832
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5837,
                "end": 5847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5837,
              "end": 5847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5852,
                "end": 5864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5852,
              "end": 5864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5869,
                "end": 5872
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5883,
                      "end": 5893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5883,
                    "end": 5893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5902,
                      "end": 5909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5902,
                    "end": 5909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5918,
                      "end": 5927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5918,
                    "end": 5927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5936,
                      "end": 5945
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5936,
                    "end": 5945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5954,
                      "end": 5963
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5954,
                    "end": 5963
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5972,
                      "end": 5978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5972,
                    "end": 5978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5987,
                      "end": 5994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5987,
                    "end": 5994
                  }
                }
              ],
              "loc": {
                "start": 5873,
                "end": 6000
              }
            },
            "loc": {
              "start": 5869,
              "end": 6000
            }
          }
        ],
        "loc": {
          "start": 5511,
          "end": 6002
        }
      },
      "loc": {
        "start": 5502,
        "end": 6002
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6003,
          "end": 6005
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6003,
        "end": 6005
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6006,
          "end": 6016
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6006,
        "end": 6016
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6017,
          "end": 6027
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6017,
        "end": 6027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6028,
          "end": 6037
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6028,
        "end": 6037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6038,
          "end": 6049
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6038,
        "end": 6049
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6050,
          "end": 6056
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 6066,
                "end": 6076
              }
            },
            "directives": [],
            "loc": {
              "start": 6063,
              "end": 6076
            }
          }
        ],
        "loc": {
          "start": 6057,
          "end": 6078
        }
      },
      "loc": {
        "start": 6050,
        "end": 6078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6079,
          "end": 6084
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 6098,
                  "end": 6110
                }
              },
              "loc": {
                "start": 6098,
                "end": 6110
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6124,
                      "end": 6140
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6121,
                    "end": 6140
                  }
                }
              ],
              "loc": {
                "start": 6111,
                "end": 6146
              }
            },
            "loc": {
              "start": 6091,
              "end": 6146
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6158,
                  "end": 6162
                }
              },
              "loc": {
                "start": 6158,
                "end": 6162
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6176,
                      "end": 6184
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6173,
                    "end": 6184
                  }
                }
              ],
              "loc": {
                "start": 6163,
                "end": 6190
              }
            },
            "loc": {
              "start": 6151,
              "end": 6190
            }
          }
        ],
        "loc": {
          "start": 6085,
          "end": 6192
        }
      },
      "loc": {
        "start": 6079,
        "end": 6192
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6193,
          "end": 6204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6193,
        "end": 6204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6205,
          "end": 6219
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6205,
        "end": 6219
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6220,
          "end": 6225
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6220,
        "end": 6225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6226,
          "end": 6235
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6226,
        "end": 6235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6236,
          "end": 6240
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6250,
                "end": 6258
              }
            },
            "directives": [],
            "loc": {
              "start": 6247,
              "end": 6258
            }
          }
        ],
        "loc": {
          "start": 6241,
          "end": 6260
        }
      },
      "loc": {
        "start": 6236,
        "end": 6260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6261,
          "end": 6275
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6261,
        "end": 6275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6276,
          "end": 6281
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6276,
        "end": 6281
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6282,
          "end": 6285
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6292,
                "end": 6301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6292,
              "end": 6301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6306,
                "end": 6317
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6306,
              "end": 6317
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6322,
                "end": 6333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6322,
              "end": 6333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6338,
                "end": 6347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6338,
              "end": 6347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6352,
                "end": 6359
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6352,
              "end": 6359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6364,
                "end": 6372
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6364,
              "end": 6372
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6377,
                "end": 6389
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6377,
              "end": 6389
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6394,
                "end": 6402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6394,
              "end": 6402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6407,
                "end": 6415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6407,
              "end": 6415
            }
          }
        ],
        "loc": {
          "start": 6286,
          "end": 6417
        }
      },
      "loc": {
        "start": 6282,
        "end": 6417
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6466,
          "end": 6468
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6466,
        "end": 6468
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6469,
          "end": 6478
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6469,
        "end": 6478
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6518,
          "end": 6526
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6533,
                "end": 6545
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6556,
                      "end": 6558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6556,
                    "end": 6558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6567,
                      "end": 6575
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6567,
                    "end": 6575
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6584,
                      "end": 6595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6584,
                    "end": 6595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6604,
                      "end": 6616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6604,
                    "end": 6616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6625,
                      "end": 6629
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6625,
                    "end": 6629
                  }
                }
              ],
              "loc": {
                "start": 6546,
                "end": 6635
              }
            },
            "loc": {
              "start": 6533,
              "end": 6635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6640,
                "end": 6642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6640,
              "end": 6642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6647,
                "end": 6657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6647,
              "end": 6657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6662,
                "end": 6672
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6662,
              "end": 6672
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6677,
                "end": 6687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6677,
              "end": 6687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6692,
                "end": 6698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6692,
              "end": 6698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6703,
                "end": 6711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6703,
              "end": 6711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6716,
                "end": 6725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6716,
              "end": 6725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6730,
                "end": 6737
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6730,
              "end": 6737
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6742,
                "end": 6754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6742,
              "end": 6754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6759,
                "end": 6764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6759,
              "end": 6764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6769,
                "end": 6772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6769,
              "end": 6772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6777,
                "end": 6789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6777,
              "end": 6789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6794,
                "end": 6806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6794,
              "end": 6806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6811,
                "end": 6824
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6811,
              "end": 6824
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6829,
                "end": 6851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6829,
              "end": 6851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6856,
                "end": 6866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6856,
              "end": 6866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6871,
                "end": 6883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6871,
              "end": 6883
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6888,
                "end": 6891
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 6902,
                      "end": 6912
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6902,
                    "end": 6912
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6921,
                      "end": 6928
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6921,
                    "end": 6928
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6937,
                      "end": 6946
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6937,
                    "end": 6946
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6955,
                      "end": 6964
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6955,
                    "end": 6964
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6973,
                      "end": 6982
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6973,
                    "end": 6982
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6991,
                      "end": 6997
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6991,
                    "end": 6997
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7006,
                      "end": 7013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7006,
                    "end": 7013
                  }
                }
              ],
              "loc": {
                "start": 6892,
                "end": 7019
              }
            },
            "loc": {
              "start": 6888,
              "end": 7019
            }
          }
        ],
        "loc": {
          "start": 6527,
          "end": 7021
        }
      },
      "loc": {
        "start": 6518,
        "end": 7021
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7022,
          "end": 7024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7022,
        "end": 7024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7025,
          "end": 7035
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7025,
        "end": 7035
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7036,
          "end": 7046
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7036,
        "end": 7046
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7047,
          "end": 7056
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7047,
        "end": 7056
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 7057,
          "end": 7068
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7057,
        "end": 7068
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 7069,
          "end": 7075
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 7085,
                "end": 7095
              }
            },
            "directives": [],
            "loc": {
              "start": 7082,
              "end": 7095
            }
          }
        ],
        "loc": {
          "start": 7076,
          "end": 7097
        }
      },
      "loc": {
        "start": 7069,
        "end": 7097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 7098,
          "end": 7103
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 7117,
                  "end": 7129
                }
              },
              "loc": {
                "start": 7117,
                "end": 7129
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7143,
                      "end": 7159
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7140,
                    "end": 7159
                  }
                }
              ],
              "loc": {
                "start": 7130,
                "end": 7165
              }
            },
            "loc": {
              "start": 7110,
              "end": 7165
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 7177,
                  "end": 7181
                }
              },
              "loc": {
                "start": 7177,
                "end": 7181
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7195,
                      "end": 7203
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7192,
                    "end": 7203
                  }
                }
              ],
              "loc": {
                "start": 7182,
                "end": 7209
              }
            },
            "loc": {
              "start": 7170,
              "end": 7209
            }
          }
        ],
        "loc": {
          "start": 7104,
          "end": 7211
        }
      },
      "loc": {
        "start": 7098,
        "end": 7211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7212,
          "end": 7223
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7212,
        "end": 7223
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7224,
          "end": 7238
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7224,
        "end": 7238
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7239,
          "end": 7244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7239,
        "end": 7244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7245,
          "end": 7254
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7245,
        "end": 7254
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7255,
          "end": 7259
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 7269,
                "end": 7277
              }
            },
            "directives": [],
            "loc": {
              "start": 7266,
              "end": 7277
            }
          }
        ],
        "loc": {
          "start": 7260,
          "end": 7279
        }
      },
      "loc": {
        "start": 7255,
        "end": 7279
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7280,
          "end": 7294
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7280,
        "end": 7294
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7295,
          "end": 7300
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7295,
        "end": 7300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7301,
          "end": 7304
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7311,
                "end": 7320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7311,
              "end": 7320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7325,
                "end": 7336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7325,
              "end": 7336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7341,
                "end": 7352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7341,
              "end": 7352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7357,
                "end": 7366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7357,
              "end": 7366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7371,
                "end": 7378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7371,
              "end": 7378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7383,
                "end": 7391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7383,
              "end": 7391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7396,
                "end": 7408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7396,
              "end": 7408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7413,
                "end": 7421
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7413,
              "end": 7421
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7426,
                "end": 7434
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7426,
              "end": 7434
            }
          }
        ],
        "loc": {
          "start": 7305,
          "end": 7436
        }
      },
      "loc": {
        "start": 7301,
        "end": 7436
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7475,
          "end": 7477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7475,
        "end": 7477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7478,
          "end": 7487
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7478,
        "end": 7487
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7517,
          "end": 7519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7517,
        "end": 7519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7520,
          "end": 7530
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7520,
        "end": 7530
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 7531,
          "end": 7534
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7531,
        "end": 7534
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7535,
          "end": 7544
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7535,
        "end": 7544
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7545,
          "end": 7557
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7564,
                "end": 7566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7564,
              "end": 7566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7571,
                "end": 7579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7571,
              "end": 7579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 7584,
                "end": 7595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7584,
              "end": 7595
            }
          }
        ],
        "loc": {
          "start": 7558,
          "end": 7597
        }
      },
      "loc": {
        "start": 7545,
        "end": 7597
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7598,
          "end": 7601
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 7608,
                "end": 7613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7608,
              "end": 7613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7618,
                "end": 7630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7618,
              "end": 7630
            }
          }
        ],
        "loc": {
          "start": 7602,
          "end": 7632
        }
      },
      "loc": {
        "start": 7598,
        "end": 7632
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7664,
          "end": 7676
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7683,
                "end": 7685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7683,
              "end": 7685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7690,
                "end": 7698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7690,
              "end": 7698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7703,
                "end": 7706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7703,
              "end": 7706
            }
          }
        ],
        "loc": {
          "start": 7677,
          "end": 7708
        }
      },
      "loc": {
        "start": 7664,
        "end": 7708
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7709,
          "end": 7711
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7709,
        "end": 7711
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7712,
          "end": 7722
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7712,
        "end": 7722
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7723,
          "end": 7733
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7723,
        "end": 7733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7734,
          "end": 7745
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7734,
        "end": 7745
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7746,
          "end": 7752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7746,
        "end": 7752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7753,
          "end": 7758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7753,
        "end": 7758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7759,
          "end": 7779
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7759,
        "end": 7779
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7780,
          "end": 7784
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7780,
        "end": 7784
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7785,
          "end": 7797
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7785,
        "end": 7797
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7798,
          "end": 7807
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7798,
        "end": 7807
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7808,
          "end": 7828
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7808,
        "end": 7828
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7829,
          "end": 7832
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7839,
                "end": 7848
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7839,
              "end": 7848
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7853,
                "end": 7862
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7853,
              "end": 7862
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7867,
                "end": 7876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7867,
              "end": 7876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7881,
                "end": 7893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7881,
              "end": 7893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7898,
                "end": 7906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7898,
              "end": 7906
            }
          }
        ],
        "loc": {
          "start": 7833,
          "end": 7908
        }
      },
      "loc": {
        "start": 7829,
        "end": 7908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7939,
          "end": 7941
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7939,
        "end": 7941
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7942,
          "end": 7952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7942,
        "end": 7952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7953,
          "end": 7963
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7953,
        "end": 7963
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7964,
          "end": 7975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7964,
        "end": 7975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7976,
          "end": 7982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7976,
        "end": 7982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7983,
          "end": 7988
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7983,
        "end": 7988
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7989,
          "end": 8009
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7989,
        "end": 8009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8010,
          "end": 8014
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8010,
        "end": 8014
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8015,
          "end": 8027
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8015,
        "end": 8027
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 947,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 950,
                "end": 960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 961,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 961,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 972,
                "end": 977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 978,
                "end": 983
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 978,
              "end": 983
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 984,
                "end": 989
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1003,
                        "end": 1015
                      }
                    },
                    "loc": {
                      "start": 1003,
                      "end": 1015
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1029,
                            "end": 1045
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1026,
                          "end": 1045
                        }
                      }
                    ],
                    "loc": {
                      "start": 1016,
                      "end": 1051
                    }
                  },
                  "loc": {
                    "start": 996,
                    "end": 1051
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1063,
                        "end": 1067
                      }
                    },
                    "loc": {
                      "start": 1063,
                      "end": 1067
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1081,
                            "end": 1089
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1078,
                          "end": 1089
                        }
                      }
                    ],
                    "loc": {
                      "start": 1068,
                      "end": 1095
                    }
                  },
                  "loc": {
                    "start": 1056,
                    "end": 1095
                  }
                }
              ],
              "loc": {
                "start": 990,
                "end": 1097
              }
            },
            "loc": {
              "start": 984,
              "end": 1097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1098,
                "end": 1101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1108,
                      "end": 1117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1108,
                    "end": 1117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1122,
                      "end": 1131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1131
                  }
                }
              ],
              "loc": {
                "start": 1102,
                "end": 1133
              }
            },
            "loc": {
              "start": 1098,
              "end": 1133
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1135
        }
      },
      "loc": {
        "start": 916,
        "end": 1135
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1145,
          "end": 1154
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1158,
            "end": 1162
          }
        },
        "loc": {
          "start": 1158,
          "end": 1162
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1165,
                "end": 1173
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1180,
                      "end": 1192
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1203,
                            "end": 1205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1203,
                          "end": 1205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1214,
                            "end": 1222
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1214,
                          "end": 1222
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1231,
                            "end": 1242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1242
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1251,
                            "end": 1255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1251,
                          "end": 1255
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 1264,
                            "end": 1269
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1284,
                                  "end": 1286
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1284,
                                "end": 1286
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 1299,
                                  "end": 1308
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1299,
                                "end": 1308
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 1321,
                                  "end": 1325
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1321,
                                "end": 1325
                              }
                            }
                          ],
                          "loc": {
                            "start": 1270,
                            "end": 1335
                          }
                        },
                        "loc": {
                          "start": 1264,
                          "end": 1335
                        }
                      }
                    ],
                    "loc": {
                      "start": 1193,
                      "end": 1341
                    }
                  },
                  "loc": {
                    "start": 1180,
                    "end": 1341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1346,
                      "end": 1348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1353,
                      "end": 1363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1353,
                    "end": 1363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1368,
                      "end": 1378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1368,
                    "end": 1378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1383,
                      "end": 1391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1383,
                    "end": 1391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1396,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1396,
                    "end": 1405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1410,
                      "end": 1422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1410,
                    "end": 1422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1427,
                      "end": 1439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1427,
                    "end": 1439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1444,
                      "end": 1456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1444,
                    "end": 1456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1461,
                      "end": 1464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 1475,
                            "end": 1485
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1475,
                          "end": 1485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1494,
                            "end": 1501
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1494,
                          "end": 1501
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1510,
                            "end": 1519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1510,
                          "end": 1519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1528,
                            "end": 1537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1528,
                          "end": 1537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1546,
                            "end": 1555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1546,
                          "end": 1555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1564,
                            "end": 1570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1564,
                          "end": 1570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1579,
                            "end": 1586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1579,
                          "end": 1586
                        }
                      }
                    ],
                    "loc": {
                      "start": 1465,
                      "end": 1592
                    }
                  },
                  "loc": {
                    "start": 1461,
                    "end": 1592
                  }
                }
              ],
              "loc": {
                "start": 1174,
                "end": 1594
              }
            },
            "loc": {
              "start": 1165,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1595,
                "end": 1597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1595,
              "end": 1597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1598,
                "end": 1608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1598,
              "end": 1608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1609,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1609,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1620,
                "end": 1629
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1620,
              "end": 1629
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1630,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1630,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1642,
                "end": 1648
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1658,
                      "end": 1668
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1655,
                    "end": 1668
                  }
                }
              ],
              "loc": {
                "start": 1649,
                "end": 1670
              }
            },
            "loc": {
              "start": 1642,
              "end": 1670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1671,
                "end": 1676
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1690,
                        "end": 1702
                      }
                    },
                    "loc": {
                      "start": 1690,
                      "end": 1702
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1716,
                            "end": 1732
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1713,
                          "end": 1732
                        }
                      }
                    ],
                    "loc": {
                      "start": 1703,
                      "end": 1738
                    }
                  },
                  "loc": {
                    "start": 1683,
                    "end": 1738
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1750,
                        "end": 1754
                      }
                    },
                    "loc": {
                      "start": 1750,
                      "end": 1754
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1768,
                            "end": 1776
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1765,
                          "end": 1776
                        }
                      }
                    ],
                    "loc": {
                      "start": 1755,
                      "end": 1782
                    }
                  },
                  "loc": {
                    "start": 1743,
                    "end": 1782
                  }
                }
              ],
              "loc": {
                "start": 1677,
                "end": 1784
              }
            },
            "loc": {
              "start": 1671,
              "end": 1784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1785,
                "end": 1796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1785,
              "end": 1796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1797,
                "end": 1811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1797,
              "end": 1811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1812,
                "end": 1817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1812,
              "end": 1817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1818,
                "end": 1827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1818,
              "end": 1827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1828,
                "end": 1832
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1842,
                      "end": 1850
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1839,
                    "end": 1850
                  }
                }
              ],
              "loc": {
                "start": 1833,
                "end": 1852
              }
            },
            "loc": {
              "start": 1828,
              "end": 1852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1853,
                "end": 1867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1853,
              "end": 1867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1868,
                "end": 1873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1868,
              "end": 1873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1874,
                "end": 1877
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1884,
                      "end": 1893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1884,
                    "end": 1893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1898,
                      "end": 1909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1898,
                    "end": 1909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1914,
                      "end": 1925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1914,
                    "end": 1925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1930,
                      "end": 1939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1930,
                    "end": 1939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1944,
                      "end": 1951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1944,
                    "end": 1951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1956,
                      "end": 1964
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1956,
                    "end": 1964
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1969,
                      "end": 1981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1969,
                    "end": 1981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1986,
                      "end": 1994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1986,
                    "end": 1994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1999,
                      "end": 2007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1999,
                    "end": 2007
                  }
                }
              ],
              "loc": {
                "start": 1878,
                "end": 2009
              }
            },
            "loc": {
              "start": 1874,
              "end": 2009
            }
          }
        ],
        "loc": {
          "start": 1163,
          "end": 2011
        }
      },
      "loc": {
        "start": 1136,
        "end": 2011
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2021,
          "end": 2029
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2033,
            "end": 2037
          }
        },
        "loc": {
          "start": 2033,
          "end": 2037
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2040,
                "end": 2042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2040,
              "end": 2042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2043,
                "end": 2052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2052
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2054
        }
      },
      "loc": {
        "start": 2012,
        "end": 2054
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2064,
          "end": 2081
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2085,
            "end": 2097
          }
        },
        "loc": {
          "start": 2085,
          "end": 2097
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2100,
                "end": 2102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2100,
              "end": 2102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2103,
                "end": 2114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2103,
              "end": 2114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2115,
                "end": 2121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2115,
              "end": 2121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2122,
                "end": 2132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2122,
              "end": 2132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2133,
                "end": 2143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2144,
                "end": 2162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2163,
                "end": 2172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2163,
              "end": 2172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2173,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2187,
                "end": 2199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2187,
              "end": 2199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2200,
                "end": 2212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2200,
              "end": 2212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2213,
                "end": 2225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2213,
              "end": 2225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2226,
                "end": 2235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2226,
              "end": 2235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2236,
                "end": 2240
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2250,
                      "end": 2258
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2247,
                    "end": 2258
                  }
                }
              ],
              "loc": {
                "start": 2241,
                "end": 2260
              }
            },
            "loc": {
              "start": 2236,
              "end": 2260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2261,
                "end": 2273
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2280,
                      "end": 2282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2280,
                    "end": 2282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2287,
                      "end": 2295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2287,
                    "end": 2295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2300,
                      "end": 2303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2300,
                    "end": 2303
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2308,
                      "end": 2312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2308,
                    "end": 2312
                  }
                }
              ],
              "loc": {
                "start": 2274,
                "end": 2314
              }
            },
            "loc": {
              "start": 2261,
              "end": 2314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2315,
                "end": 2318
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2325,
                      "end": 2338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2325,
                    "end": 2338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2343,
                      "end": 2352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2343,
                    "end": 2352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2357,
                      "end": 2368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2357,
                    "end": 2368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2373,
                      "end": 2382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2373,
                    "end": 2382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2387,
                      "end": 2396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2387,
                    "end": 2396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2401,
                      "end": 2408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2401,
                    "end": 2408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2413,
                      "end": 2425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2413,
                    "end": 2425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2430,
                      "end": 2438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2430,
                    "end": 2438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2443,
                      "end": 2457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2468,
                            "end": 2470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2468,
                          "end": 2470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2479,
                            "end": 2489
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2479,
                          "end": 2489
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2498,
                            "end": 2508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2498,
                          "end": 2508
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2517,
                            "end": 2524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2517,
                          "end": 2524
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2533,
                            "end": 2544
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2533,
                          "end": 2544
                        }
                      }
                    ],
                    "loc": {
                      "start": 2458,
                      "end": 2550
                    }
                  },
                  "loc": {
                    "start": 2443,
                    "end": 2550
                  }
                }
              ],
              "loc": {
                "start": 2319,
                "end": 2552
              }
            },
            "loc": {
              "start": 2315,
              "end": 2552
            }
          }
        ],
        "loc": {
          "start": 2098,
          "end": 2554
        }
      },
      "loc": {
        "start": 2055,
        "end": 2554
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2564,
          "end": 2580
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2584,
            "end": 2596
          }
        },
        "loc": {
          "start": 2584,
          "end": 2596
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2599,
                "end": 2601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2599,
              "end": 2601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2602,
                "end": 2613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2602,
              "end": 2613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2614,
                "end": 2620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2614,
              "end": 2620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2621,
                "end": 2633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2621,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2634,
                "end": 2637
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2644,
                      "end": 2657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2644,
                    "end": 2657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2662,
                      "end": 2671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2662,
                    "end": 2671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2676,
                      "end": 2687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2676,
                    "end": 2687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2692,
                      "end": 2701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2692,
                    "end": 2701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2706,
                      "end": 2715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2706,
                    "end": 2715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2720,
                      "end": 2727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2720,
                    "end": 2727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2732,
                      "end": 2744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2732,
                    "end": 2744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2749,
                      "end": 2757
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2749,
                    "end": 2757
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2762,
                      "end": 2776
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2787,
                            "end": 2789
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2787,
                          "end": 2789
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2798,
                            "end": 2808
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2798,
                          "end": 2808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2817,
                            "end": 2827
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2817,
                          "end": 2827
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2836,
                            "end": 2843
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2836,
                          "end": 2843
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2852,
                            "end": 2863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2852,
                          "end": 2863
                        }
                      }
                    ],
                    "loc": {
                      "start": 2777,
                      "end": 2869
                    }
                  },
                  "loc": {
                    "start": 2762,
                    "end": 2869
                  }
                }
              ],
              "loc": {
                "start": 2638,
                "end": 2871
              }
            },
            "loc": {
              "start": 2634,
              "end": 2871
            }
          }
        ],
        "loc": {
          "start": 2597,
          "end": 2873
        }
      },
      "loc": {
        "start": 2555,
        "end": 2873
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2883,
          "end": 2895
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 2899,
            "end": 2906
          }
        },
        "loc": {
          "start": 2899,
          "end": 2906
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2909,
                "end": 2917
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2924,
                      "end": 2936
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2947,
                            "end": 2949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2947,
                          "end": 2949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2958,
                            "end": 2966
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2958,
                          "end": 2966
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2975,
                            "end": 2986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2975,
                          "end": 2986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2995,
                            "end": 2999
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2995,
                          "end": 2999
                        }
                      }
                    ],
                    "loc": {
                      "start": 2937,
                      "end": 3005
                    }
                  },
                  "loc": {
                    "start": 2924,
                    "end": 3005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3010,
                      "end": 3012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3010,
                    "end": 3012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3017,
                      "end": 3027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3017,
                    "end": 3027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3032,
                      "end": 3042
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3032,
                    "end": 3042
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3047,
                      "end": 3063
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3047,
                    "end": 3063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3068,
                      "end": 3076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3068,
                    "end": 3076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3081,
                      "end": 3090
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3081,
                    "end": 3090
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3095,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3095,
                    "end": 3107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3112,
                      "end": 3128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3112,
                    "end": 3128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3133,
                      "end": 3143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3133,
                    "end": 3143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3148,
                      "end": 3160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3148,
                    "end": 3160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3165,
                      "end": 3177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3165,
                    "end": 3177
                  }
                }
              ],
              "loc": {
                "start": 2918,
                "end": 3179
              }
            },
            "loc": {
              "start": 2909,
              "end": 3179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3180,
                "end": 3182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3180,
              "end": 3182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3183,
                "end": 3193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3183,
              "end": 3193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3194,
                "end": 3204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3194,
              "end": 3204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3205,
                "end": 3214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3205,
              "end": 3214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3215,
                "end": 3226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3215,
              "end": 3226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3227,
                "end": 3233
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3243,
                      "end": 3253
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3240,
                    "end": 3253
                  }
                }
              ],
              "loc": {
                "start": 3234,
                "end": 3255
              }
            },
            "loc": {
              "start": 3227,
              "end": 3255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3256,
                "end": 3261
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 3275,
                        "end": 3287
                      }
                    },
                    "loc": {
                      "start": 3275,
                      "end": 3287
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3301,
                            "end": 3317
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3298,
                          "end": 3317
                        }
                      }
                    ],
                    "loc": {
                      "start": 3288,
                      "end": 3323
                    }
                  },
                  "loc": {
                    "start": 3268,
                    "end": 3323
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3335,
                        "end": 3339
                      }
                    },
                    "loc": {
                      "start": 3335,
                      "end": 3339
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3353,
                            "end": 3361
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3350,
                          "end": 3361
                        }
                      }
                    ],
                    "loc": {
                      "start": 3340,
                      "end": 3367
                    }
                  },
                  "loc": {
                    "start": 3328,
                    "end": 3367
                  }
                }
              ],
              "loc": {
                "start": 3262,
                "end": 3369
              }
            },
            "loc": {
              "start": 3256,
              "end": 3369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3370,
                "end": 3381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3370,
              "end": 3381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3382,
                "end": 3396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3382,
              "end": 3396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3397,
                "end": 3402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3397,
              "end": 3402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3403,
                "end": 3412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3403,
              "end": 3412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3413,
                "end": 3417
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 3427,
                      "end": 3435
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3424,
                    "end": 3435
                  }
                }
              ],
              "loc": {
                "start": 3418,
                "end": 3437
              }
            },
            "loc": {
              "start": 3413,
              "end": 3437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3438,
                "end": 3452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3438,
              "end": 3452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3453,
                "end": 3458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3453,
              "end": 3458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3459,
                "end": 3462
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 3469,
                      "end": 3478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3469,
                    "end": 3478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3483,
                      "end": 3494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3483,
                    "end": 3494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3499,
                      "end": 3510
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3499,
                    "end": 3510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3515,
                      "end": 3524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3515,
                    "end": 3524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3529,
                      "end": 3536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3529,
                    "end": 3536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3541,
                      "end": 3549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3541,
                    "end": 3549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3554,
                      "end": 3566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3554,
                    "end": 3566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3571,
                      "end": 3579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3571,
                    "end": 3579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3584,
                      "end": 3592
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3584,
                    "end": 3592
                  }
                }
              ],
              "loc": {
                "start": 3463,
                "end": 3594
              }
            },
            "loc": {
              "start": 3459,
              "end": 3594
            }
          }
        ],
        "loc": {
          "start": 2907,
          "end": 3596
        }
      },
      "loc": {
        "start": 2874,
        "end": 3596
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3606,
          "end": 3617
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3621,
            "end": 3628
          }
        },
        "loc": {
          "start": 3621,
          "end": 3628
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3631,
                "end": 3633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3631,
              "end": 3633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3634,
                "end": 3643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3634,
              "end": 3643
            }
          }
        ],
        "loc": {
          "start": 3629,
          "end": 3645
        }
      },
      "loc": {
        "start": 3597,
        "end": 3645
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3655,
          "end": 3668
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3672,
            "end": 3680
          }
        },
        "loc": {
          "start": 3672,
          "end": 3680
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3683,
                "end": 3695
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3702,
                      "end": 3704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3702,
                    "end": 3704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3709,
                      "end": 3717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3709,
                    "end": 3717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3722,
                      "end": 3733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3722,
                    "end": 3733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3738,
                      "end": 3742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3738,
                    "end": 3742
                  }
                }
              ],
              "loc": {
                "start": 3696,
                "end": 3744
              }
            },
            "loc": {
              "start": 3683,
              "end": 3744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3745,
                "end": 3747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3745,
              "end": 3747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3748,
                "end": 3758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3748,
              "end": 3758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3759,
                "end": 3769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3759,
              "end": 3769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3770,
                "end": 3779
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3786,
                      "end": 3788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3786,
                    "end": 3788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3793,
                      "end": 3803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3793,
                    "end": 3803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3808,
                      "end": 3818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3808,
                    "end": 3818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3823,
                      "end": 3834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3823,
                    "end": 3834
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3839,
                      "end": 3845
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3839,
                    "end": 3845
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3850,
                      "end": 3855
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3850,
                    "end": 3855
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3860,
                      "end": 3880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3860,
                    "end": 3880
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3885,
                      "end": 3889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3885,
                    "end": 3889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3894,
                      "end": 3906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3894,
                    "end": 3906
                  }
                }
              ],
              "loc": {
                "start": 3780,
                "end": 3908
              }
            },
            "loc": {
              "start": 3770,
              "end": 3908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3909,
                "end": 3926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3909,
              "end": 3926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3927,
                "end": 3936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3927,
              "end": 3936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3937,
                "end": 3942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3937,
              "end": 3942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3943,
                "end": 3952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3943,
              "end": 3952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 3953,
                "end": 3965
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3953,
              "end": 3965
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 3966,
                "end": 3979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3966,
              "end": 3979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3980,
                "end": 3992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3980,
              "end": 3992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 3993,
                "end": 4002
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4016,
                        "end": 4019
                      }
                    },
                    "loc": {
                      "start": 4016,
                      "end": 4019
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4033,
                            "end": 4040
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4030,
                          "end": 4040
                        }
                      }
                    ],
                    "loc": {
                      "start": 4020,
                      "end": 4046
                    }
                  },
                  "loc": {
                    "start": 4009,
                    "end": 4046
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4058,
                        "end": 4062
                      }
                    },
                    "loc": {
                      "start": 4058,
                      "end": 4062
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4076,
                            "end": 4084
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4073,
                          "end": 4084
                        }
                      }
                    ],
                    "loc": {
                      "start": 4063,
                      "end": 4090
                    }
                  },
                  "loc": {
                    "start": 4051,
                    "end": 4090
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4102,
                        "end": 4114
                      }
                    },
                    "loc": {
                      "start": 4102,
                      "end": 4114
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4128,
                            "end": 4144
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4125,
                          "end": 4144
                        }
                      }
                    ],
                    "loc": {
                      "start": 4115,
                      "end": 4150
                    }
                  },
                  "loc": {
                    "start": 4095,
                    "end": 4150
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4162,
                        "end": 4169
                      }
                    },
                    "loc": {
                      "start": 4162,
                      "end": 4169
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4183,
                            "end": 4194
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4180,
                          "end": 4194
                        }
                      }
                    ],
                    "loc": {
                      "start": 4170,
                      "end": 4200
                    }
                  },
                  "loc": {
                    "start": 4155,
                    "end": 4200
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4212,
                        "end": 4219
                      }
                    },
                    "loc": {
                      "start": 4212,
                      "end": 4219
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4233,
                            "end": 4244
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4230,
                          "end": 4244
                        }
                      }
                    ],
                    "loc": {
                      "start": 4220,
                      "end": 4250
                    }
                  },
                  "loc": {
                    "start": 4205,
                    "end": 4250
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4262,
                        "end": 4275
                      }
                    },
                    "loc": {
                      "start": 4262,
                      "end": 4275
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4289,
                            "end": 4306
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4286,
                          "end": 4306
                        }
                      }
                    ],
                    "loc": {
                      "start": 4276,
                      "end": 4312
                    }
                  },
                  "loc": {
                    "start": 4255,
                    "end": 4312
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4324,
                        "end": 4332
                      }
                    },
                    "loc": {
                      "start": 4324,
                      "end": 4332
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4346,
                            "end": 4358
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4343,
                          "end": 4358
                        }
                      }
                    ],
                    "loc": {
                      "start": 4333,
                      "end": 4364
                    }
                  },
                  "loc": {
                    "start": 4317,
                    "end": 4364
                  }
                }
              ],
              "loc": {
                "start": 4003,
                "end": 4366
              }
            },
            "loc": {
              "start": 3993,
              "end": 4366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4367,
                "end": 4371
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 4381,
                      "end": 4389
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4378,
                    "end": 4389
                  }
                }
              ],
              "loc": {
                "start": 4372,
                "end": 4391
              }
            },
            "loc": {
              "start": 4367,
              "end": 4391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4392,
                "end": 4395
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4402,
                      "end": 4410
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4402,
                    "end": 4410
                  }
                }
              ],
              "loc": {
                "start": 4396,
                "end": 4412
              }
            },
            "loc": {
              "start": 4392,
              "end": 4412
            }
          }
        ],
        "loc": {
          "start": 3681,
          "end": 4414
        }
      },
      "loc": {
        "start": 3646,
        "end": 4414
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4424,
          "end": 4436
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4440,
            "end": 4447
          }
        },
        "loc": {
          "start": 4440,
          "end": 4447
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4450,
                "end": 4458
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4465,
                      "end": 4477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4488,
                            "end": 4490
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4488,
                          "end": 4490
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4499,
                            "end": 4507
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4499,
                          "end": 4507
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4516,
                            "end": 4527
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4516,
                          "end": 4527
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4536,
                            "end": 4548
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4536,
                          "end": 4548
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4557,
                            "end": 4561
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4557,
                          "end": 4561
                        }
                      }
                    ],
                    "loc": {
                      "start": 4478,
                      "end": 4567
                    }
                  },
                  "loc": {
                    "start": 4465,
                    "end": 4567
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4572,
                      "end": 4574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4572,
                    "end": 4574
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4579,
                      "end": 4589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4579,
                    "end": 4589
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4594,
                      "end": 4604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4594,
                    "end": 4604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4609,
                      "end": 4620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4609,
                    "end": 4620
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4625,
                      "end": 4638
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4625,
                    "end": 4638
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4643,
                      "end": 4653
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4643,
                    "end": 4653
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4658,
                      "end": 4667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4658,
                    "end": 4667
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4672,
                      "end": 4680
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4672,
                    "end": 4680
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4685,
                      "end": 4694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4685,
                    "end": 4694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4699,
                      "end": 4709
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4699,
                    "end": 4709
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4714,
                      "end": 4726
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4714,
                    "end": 4726
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4731,
                      "end": 4745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4731,
                    "end": 4745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4750,
                      "end": 4771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4750,
                    "end": 4771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4776,
                      "end": 4787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4776,
                    "end": 4787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4792,
                      "end": 4804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4792,
                    "end": 4804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4809,
                      "end": 4821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4809,
                    "end": 4821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4826,
                      "end": 4839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4826,
                    "end": 4839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4844,
                      "end": 4866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4844,
                    "end": 4866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4871,
                      "end": 4881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4871,
                    "end": 4881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4886,
                      "end": 4897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4886,
                    "end": 4897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4902,
                      "end": 4912
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4902,
                    "end": 4912
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4917,
                      "end": 4931
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4917,
                    "end": 4931
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4936,
                      "end": 4948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4936,
                    "end": 4948
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 4953,
                      "end": 4965
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4953,
                    "end": 4965
                  }
                }
              ],
              "loc": {
                "start": 4459,
                "end": 4967
              }
            },
            "loc": {
              "start": 4450,
              "end": 4967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4968,
                "end": 4970
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4968,
              "end": 4970
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4971,
                "end": 4981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4971,
              "end": 4981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4982,
                "end": 4992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4982,
              "end": 4992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4993,
                "end": 5003
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4993,
              "end": 5003
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5004,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5004,
              "end": 5013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5014,
                "end": 5025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5014,
              "end": 5025
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5026,
                "end": 5032
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 5042,
                      "end": 5052
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5039,
                    "end": 5052
                  }
                }
              ],
              "loc": {
                "start": 5033,
                "end": 5054
              }
            },
            "loc": {
              "start": 5026,
              "end": 5054
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5055,
                "end": 5060
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 5074,
                        "end": 5086
                      }
                    },
                    "loc": {
                      "start": 5074,
                      "end": 5086
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5100,
                            "end": 5116
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5097,
                          "end": 5116
                        }
                      }
                    ],
                    "loc": {
                      "start": 5087,
                      "end": 5122
                    }
                  },
                  "loc": {
                    "start": 5067,
                    "end": 5122
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5134,
                        "end": 5138
                      }
                    },
                    "loc": {
                      "start": 5134,
                      "end": 5138
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5152,
                            "end": 5160
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5149,
                          "end": 5160
                        }
                      }
                    ],
                    "loc": {
                      "start": 5139,
                      "end": 5166
                    }
                  },
                  "loc": {
                    "start": 5127,
                    "end": 5166
                  }
                }
              ],
              "loc": {
                "start": 5061,
                "end": 5168
              }
            },
            "loc": {
              "start": 5055,
              "end": 5168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5169,
                "end": 5180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5169,
              "end": 5180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5181,
                "end": 5195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5181,
              "end": 5195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5196,
                "end": 5201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5196,
              "end": 5201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5202,
                "end": 5211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5202,
              "end": 5211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5212,
                "end": 5216
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 5226,
                      "end": 5234
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5223,
                    "end": 5234
                  }
                }
              ],
              "loc": {
                "start": 5217,
                "end": 5236
              }
            },
            "loc": {
              "start": 5212,
              "end": 5236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5237,
                "end": 5251
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5237,
              "end": 5251
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5252,
                "end": 5257
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5252,
              "end": 5257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5258,
                "end": 5261
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5268,
                      "end": 5278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5268,
                    "end": 5278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5283,
                      "end": 5292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5283,
                    "end": 5292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5297,
                      "end": 5308
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5297,
                    "end": 5308
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5313,
                      "end": 5322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5313,
                    "end": 5322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5327,
                      "end": 5334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5327,
                    "end": 5334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5339,
                      "end": 5347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5339,
                    "end": 5347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5352,
                      "end": 5364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5352,
                    "end": 5364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5369,
                      "end": 5377
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5369,
                    "end": 5377
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5382,
                      "end": 5390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5382,
                    "end": 5390
                  }
                }
              ],
              "loc": {
                "start": 5262,
                "end": 5392
              }
            },
            "loc": {
              "start": 5258,
              "end": 5392
            }
          }
        ],
        "loc": {
          "start": 4448,
          "end": 5394
        }
      },
      "loc": {
        "start": 4415,
        "end": 5394
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5404,
          "end": 5415
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5419,
            "end": 5426
          }
        },
        "loc": {
          "start": 5419,
          "end": 5426
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5429,
                "end": 5431
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5429,
              "end": 5431
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5432,
                "end": 5442
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5432,
              "end": 5442
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5443,
                "end": 5452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5443,
              "end": 5452
            }
          }
        ],
        "loc": {
          "start": 5427,
          "end": 5454
        }
      },
      "loc": {
        "start": 5395,
        "end": 5454
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5464,
          "end": 5482
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5486,
            "end": 5499
          }
        },
        "loc": {
          "start": 5486,
          "end": 5499
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5502,
                "end": 5510
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5517,
                      "end": 5529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5540,
                            "end": 5542
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5540,
                          "end": 5542
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5551,
                            "end": 5559
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5551,
                          "end": 5559
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5568,
                            "end": 5579
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5568,
                          "end": 5579
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5588,
                            "end": 5600
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5588,
                          "end": 5600
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5609,
                            "end": 5613
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5609,
                          "end": 5613
                        }
                      }
                    ],
                    "loc": {
                      "start": 5530,
                      "end": 5619
                    }
                  },
                  "loc": {
                    "start": 5517,
                    "end": 5619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5624,
                      "end": 5626
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5624,
                    "end": 5626
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5631,
                      "end": 5641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5631,
                    "end": 5641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5646,
                      "end": 5656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5646,
                    "end": 5656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5661,
                      "end": 5671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5661,
                    "end": 5671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5676,
                      "end": 5685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5676,
                    "end": 5685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5690,
                      "end": 5698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5690,
                    "end": 5698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5703,
                      "end": 5712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5703,
                    "end": 5712
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5717,
                      "end": 5724
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5717,
                    "end": 5724
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 5729,
                      "end": 5741
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5729,
                    "end": 5741
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 5746,
                      "end": 5753
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5746,
                    "end": 5753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5758,
                      "end": 5770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5758,
                    "end": 5770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5775,
                      "end": 5787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5775,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5792,
                      "end": 5805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5792,
                    "end": 5805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5810,
                      "end": 5832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5810,
                    "end": 5832
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5837,
                      "end": 5847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5837,
                    "end": 5847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5852,
                      "end": 5864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5852,
                    "end": 5864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5869,
                      "end": 5872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 5883,
                            "end": 5893
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5883,
                          "end": 5893
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5902,
                            "end": 5909
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5902,
                          "end": 5909
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5918,
                            "end": 5927
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5918,
                          "end": 5927
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5936,
                            "end": 5945
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5936,
                          "end": 5945
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5954,
                            "end": 5963
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5954,
                          "end": 5963
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5972,
                            "end": 5978
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5972,
                          "end": 5978
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5987,
                            "end": 5994
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5987,
                          "end": 5994
                        }
                      }
                    ],
                    "loc": {
                      "start": 5873,
                      "end": 6000
                    }
                  },
                  "loc": {
                    "start": 5869,
                    "end": 6000
                  }
                }
              ],
              "loc": {
                "start": 5511,
                "end": 6002
              }
            },
            "loc": {
              "start": 5502,
              "end": 6002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6003,
                "end": 6005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6003,
              "end": 6005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6006,
                "end": 6016
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6006,
              "end": 6016
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6017,
                "end": 6027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6017,
              "end": 6027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6028,
                "end": 6037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6028,
              "end": 6037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6038,
                "end": 6049
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6038,
              "end": 6049
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6050,
                "end": 6056
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 6066,
                      "end": 6076
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6063,
                    "end": 6076
                  }
                }
              ],
              "loc": {
                "start": 6057,
                "end": 6078
              }
            },
            "loc": {
              "start": 6050,
              "end": 6078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6079,
                "end": 6084
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 6098,
                        "end": 6110
                      }
                    },
                    "loc": {
                      "start": 6098,
                      "end": 6110
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6124,
                            "end": 6140
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6121,
                          "end": 6140
                        }
                      }
                    ],
                    "loc": {
                      "start": 6111,
                      "end": 6146
                    }
                  },
                  "loc": {
                    "start": 6091,
                    "end": 6146
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6158,
                        "end": 6162
                      }
                    },
                    "loc": {
                      "start": 6158,
                      "end": 6162
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6176,
                            "end": 6184
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6173,
                          "end": 6184
                        }
                      }
                    ],
                    "loc": {
                      "start": 6163,
                      "end": 6190
                    }
                  },
                  "loc": {
                    "start": 6151,
                    "end": 6190
                  }
                }
              ],
              "loc": {
                "start": 6085,
                "end": 6192
              }
            },
            "loc": {
              "start": 6079,
              "end": 6192
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6193,
                "end": 6204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6193,
              "end": 6204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6205,
                "end": 6219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6205,
              "end": 6219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6220,
                "end": 6225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6220,
              "end": 6225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6226,
                "end": 6235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6226,
              "end": 6235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6236,
                "end": 6240
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6250,
                      "end": 6258
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6247,
                    "end": 6258
                  }
                }
              ],
              "loc": {
                "start": 6241,
                "end": 6260
              }
            },
            "loc": {
              "start": 6236,
              "end": 6260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6261,
                "end": 6275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6261,
              "end": 6275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6276,
                "end": 6281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6276,
              "end": 6281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6282,
                "end": 6285
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6292,
                      "end": 6301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6292,
                    "end": 6301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6306,
                      "end": 6317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6306,
                    "end": 6317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6322,
                      "end": 6333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6322,
                    "end": 6333
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6338,
                      "end": 6347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6338,
                    "end": 6347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6352,
                      "end": 6359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6352,
                    "end": 6359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6364,
                      "end": 6372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6364,
                    "end": 6372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6377,
                      "end": 6389
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6377,
                    "end": 6389
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6394,
                      "end": 6402
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6394,
                    "end": 6402
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6407,
                      "end": 6415
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6407,
                    "end": 6415
                  }
                }
              ],
              "loc": {
                "start": 6286,
                "end": 6417
              }
            },
            "loc": {
              "start": 6282,
              "end": 6417
            }
          }
        ],
        "loc": {
          "start": 5500,
          "end": 6419
        }
      },
      "loc": {
        "start": 5455,
        "end": 6419
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6429,
          "end": 6446
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6450,
            "end": 6463
          }
        },
        "loc": {
          "start": 6450,
          "end": 6463
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6466,
                "end": 6468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6466,
              "end": 6468
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6469,
                "end": 6478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6469,
              "end": 6478
            }
          }
        ],
        "loc": {
          "start": 6464,
          "end": 6480
        }
      },
      "loc": {
        "start": 6420,
        "end": 6480
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6490,
          "end": 6503
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6507,
            "end": 6515
          }
        },
        "loc": {
          "start": 6507,
          "end": 6515
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 6518,
                "end": 6526
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6533,
                      "end": 6545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6556,
                            "end": 6558
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6556,
                          "end": 6558
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6567,
                            "end": 6575
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6567,
                          "end": 6575
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6584,
                            "end": 6595
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6584,
                          "end": 6595
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6604,
                            "end": 6616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6604,
                          "end": 6616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6625,
                            "end": 6629
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6625,
                          "end": 6629
                        }
                      }
                    ],
                    "loc": {
                      "start": 6546,
                      "end": 6635
                    }
                  },
                  "loc": {
                    "start": 6533,
                    "end": 6635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6640,
                      "end": 6642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6640,
                    "end": 6642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6647,
                      "end": 6657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6647,
                    "end": 6657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6662,
                      "end": 6672
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6662,
                    "end": 6672
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6677,
                      "end": 6687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6677,
                    "end": 6687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6692,
                      "end": 6698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6692,
                    "end": 6698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6703,
                      "end": 6711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6703,
                    "end": 6711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6716,
                      "end": 6725
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6716,
                    "end": 6725
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6730,
                      "end": 6737
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6730,
                    "end": 6737
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6742,
                      "end": 6754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6742,
                    "end": 6754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6759,
                      "end": 6764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6759,
                    "end": 6764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6769,
                      "end": 6772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6769,
                    "end": 6772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6777,
                      "end": 6789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6777,
                    "end": 6789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6794,
                      "end": 6806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6794,
                    "end": 6806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6811,
                      "end": 6824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6811,
                    "end": 6824
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6829,
                      "end": 6851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6829,
                    "end": 6851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6856,
                      "end": 6866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6856,
                    "end": 6866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6871,
                      "end": 6883
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6871,
                    "end": 6883
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6888,
                      "end": 6891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 6902,
                            "end": 6912
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6902,
                          "end": 6912
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6921,
                            "end": 6928
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6921,
                          "end": 6928
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6937,
                            "end": 6946
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6937,
                          "end": 6946
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6955,
                            "end": 6964
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6955,
                          "end": 6964
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6973,
                            "end": 6982
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6973,
                          "end": 6982
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6991,
                            "end": 6997
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6991,
                          "end": 6997
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7006,
                            "end": 7013
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7006,
                          "end": 7013
                        }
                      }
                    ],
                    "loc": {
                      "start": 6892,
                      "end": 7019
                    }
                  },
                  "loc": {
                    "start": 6888,
                    "end": 7019
                  }
                }
              ],
              "loc": {
                "start": 6527,
                "end": 7021
              }
            },
            "loc": {
              "start": 6518,
              "end": 7021
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7022,
                "end": 7024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7022,
              "end": 7024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7025,
                "end": 7035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7025,
              "end": 7035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7036,
                "end": 7046
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7036,
              "end": 7046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7047,
                "end": 7056
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7047,
              "end": 7056
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7057,
                "end": 7068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7057,
              "end": 7068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7069,
                "end": 7075
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 7085,
                      "end": 7095
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7082,
                    "end": 7095
                  }
                }
              ],
              "loc": {
                "start": 7076,
                "end": 7097
              }
            },
            "loc": {
              "start": 7069,
              "end": 7097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7098,
                "end": 7103
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 7117,
                        "end": 7129
                      }
                    },
                    "loc": {
                      "start": 7117,
                      "end": 7129
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7143,
                            "end": 7159
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7140,
                          "end": 7159
                        }
                      }
                    ],
                    "loc": {
                      "start": 7130,
                      "end": 7165
                    }
                  },
                  "loc": {
                    "start": 7110,
                    "end": 7165
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 7177,
                        "end": 7181
                      }
                    },
                    "loc": {
                      "start": 7177,
                      "end": 7181
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7195,
                            "end": 7203
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7192,
                          "end": 7203
                        }
                      }
                    ],
                    "loc": {
                      "start": 7182,
                      "end": 7209
                    }
                  },
                  "loc": {
                    "start": 7170,
                    "end": 7209
                  }
                }
              ],
              "loc": {
                "start": 7104,
                "end": 7211
              }
            },
            "loc": {
              "start": 7098,
              "end": 7211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7212,
                "end": 7223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7212,
              "end": 7223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7224,
                "end": 7238
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7224,
              "end": 7238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7239,
                "end": 7244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7239,
              "end": 7244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7245,
                "end": 7254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7245,
              "end": 7254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7255,
                "end": 7259
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 7269,
                      "end": 7277
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7266,
                    "end": 7277
                  }
                }
              ],
              "loc": {
                "start": 7260,
                "end": 7279
              }
            },
            "loc": {
              "start": 7255,
              "end": 7279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7280,
                "end": 7294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7280,
              "end": 7294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7295,
                "end": 7300
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7295,
              "end": 7300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7301,
                "end": 7304
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7311,
                      "end": 7320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7311,
                    "end": 7320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7325,
                      "end": 7336
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7325,
                    "end": 7336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7341,
                      "end": 7352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7341,
                    "end": 7352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7357,
                      "end": 7366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7357,
                    "end": 7366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7371,
                      "end": 7378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7371,
                    "end": 7378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7383,
                      "end": 7391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7383,
                    "end": 7391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7396,
                      "end": 7408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7396,
                    "end": 7408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7413,
                      "end": 7421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7413,
                    "end": 7421
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7426,
                      "end": 7434
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7426,
                    "end": 7434
                  }
                }
              ],
              "loc": {
                "start": 7305,
                "end": 7436
              }
            },
            "loc": {
              "start": 7301,
              "end": 7436
            }
          }
        ],
        "loc": {
          "start": 6516,
          "end": 7438
        }
      },
      "loc": {
        "start": 6481,
        "end": 7438
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7448,
          "end": 7460
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7464,
            "end": 7472
          }
        },
        "loc": {
          "start": 7464,
          "end": 7472
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7475,
                "end": 7477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7475,
              "end": 7477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7478,
                "end": 7487
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7478,
              "end": 7487
            }
          }
        ],
        "loc": {
          "start": 7473,
          "end": 7489
        }
      },
      "loc": {
        "start": 7439,
        "end": 7489
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7499,
          "end": 7507
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7511,
            "end": 7514
          }
        },
        "loc": {
          "start": 7511,
          "end": 7514
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7517,
                "end": 7519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7517,
              "end": 7519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7520,
                "end": 7530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7520,
              "end": 7530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7531,
                "end": 7534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7531,
              "end": 7534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7535,
                "end": 7544
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7535,
              "end": 7544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7545,
                "end": 7557
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7564,
                      "end": 7566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7564,
                    "end": 7566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7571,
                      "end": 7579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7571,
                    "end": 7579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7584,
                      "end": 7595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7584,
                    "end": 7595
                  }
                }
              ],
              "loc": {
                "start": 7558,
                "end": 7597
              }
            },
            "loc": {
              "start": 7545,
              "end": 7597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7598,
                "end": 7601
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 7608,
                      "end": 7613
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7608,
                    "end": 7613
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7618,
                      "end": 7630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7618,
                    "end": 7630
                  }
                }
              ],
              "loc": {
                "start": 7602,
                "end": 7632
              }
            },
            "loc": {
              "start": 7598,
              "end": 7632
            }
          }
        ],
        "loc": {
          "start": 7515,
          "end": 7634
        }
      },
      "loc": {
        "start": 7490,
        "end": 7634
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7644,
          "end": 7653
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7657,
            "end": 7661
          }
        },
        "loc": {
          "start": 7657,
          "end": 7661
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7664,
                "end": 7676
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7683,
                      "end": 7685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7683,
                    "end": 7685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7690,
                      "end": 7698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7690,
                    "end": 7698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7703,
                      "end": 7706
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7703,
                    "end": 7706
                  }
                }
              ],
              "loc": {
                "start": 7677,
                "end": 7708
              }
            },
            "loc": {
              "start": 7664,
              "end": 7708
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7709,
                "end": 7711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7709,
              "end": 7711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7712,
                "end": 7722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7712,
              "end": 7722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7723,
                "end": 7733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7723,
              "end": 7733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7734,
                "end": 7745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7734,
              "end": 7745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7746,
                "end": 7752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7746,
              "end": 7752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7753,
                "end": 7758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7753,
              "end": 7758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7759,
                "end": 7779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7759,
              "end": 7779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7780,
                "end": 7784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7780,
              "end": 7784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7785,
                "end": 7797
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7785,
              "end": 7797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7798,
                "end": 7807
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7798,
              "end": 7807
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7808,
                "end": 7828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7808,
              "end": 7828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7829,
                "end": 7832
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7839,
                      "end": 7848
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7839,
                    "end": 7848
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7853,
                      "end": 7862
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7853,
                    "end": 7862
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7867,
                      "end": 7876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7867,
                    "end": 7876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7881,
                      "end": 7893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7881,
                    "end": 7893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7898,
                      "end": 7906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7898,
                    "end": 7906
                  }
                }
              ],
              "loc": {
                "start": 7833,
                "end": 7908
              }
            },
            "loc": {
              "start": 7829,
              "end": 7908
            }
          }
        ],
        "loc": {
          "start": 7662,
          "end": 7910
        }
      },
      "loc": {
        "start": 7635,
        "end": 7910
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7920,
          "end": 7928
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7932,
            "end": 7936
          }
        },
        "loc": {
          "start": 7932,
          "end": 7936
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7939,
                "end": 7941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7939,
              "end": 7941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7942,
                "end": 7952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7942,
              "end": 7952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7953,
                "end": 7963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7953,
              "end": 7963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7964,
                "end": 7975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7964,
              "end": 7975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7976,
                "end": 7982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7976,
              "end": 7982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7983,
                "end": 7988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7983,
              "end": 7988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7989,
                "end": 8009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7989,
              "end": 8009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8010,
                "end": 8014
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8010,
              "end": 8014
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8015,
                "end": 8027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8015,
              "end": 8027
            }
          }
        ],
        "loc": {
          "start": 7937,
          "end": 8029
        }
      },
      "loc": {
        "start": 7911,
        "end": 8029
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "populars",
      "loc": {
        "start": 8037,
        "end": 8045
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8047,
              "end": 8052
            }
          },
          "loc": {
            "start": 8046,
            "end": 8052
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularSearchInput",
              "loc": {
                "start": 8054,
                "end": 8072
              }
            },
            "loc": {
              "start": 8054,
              "end": 8072
            }
          },
          "loc": {
            "start": 8054,
            "end": 8073
          }
        },
        "directives": [],
        "loc": {
          "start": 8046,
          "end": 8073
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "populars",
            "loc": {
              "start": 8079,
              "end": 8087
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8088,
                  "end": 8093
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8096,
                    "end": 8101
                  }
                },
                "loc": {
                  "start": 8095,
                  "end": 8101
                }
              },
              "loc": {
                "start": 8088,
                "end": 8101
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 8109,
                    "end": 8114
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 8125,
                          "end": 8131
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8125,
                        "end": 8131
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 8140,
                          "end": 8144
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Api",
                                "loc": {
                                  "start": 8166,
                                  "end": 8169
                                }
                              },
                              "loc": {
                                "start": 8166,
                                "end": 8169
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Api_list",
                                    "loc": {
                                      "start": 8191,
                                      "end": 8199
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8188,
                                    "end": 8199
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8170,
                                "end": 8213
                              }
                            },
                            "loc": {
                              "start": 8159,
                              "end": 8213
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Note",
                                "loc": {
                                  "start": 8233,
                                  "end": 8237
                                }
                              },
                              "loc": {
                                "start": 8233,
                                "end": 8237
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Note_list",
                                    "loc": {
                                      "start": 8259,
                                      "end": 8268
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8256,
                                    "end": 8268
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8238,
                                "end": 8282
                              }
                            },
                            "loc": {
                              "start": 8226,
                              "end": 8282
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 8302,
                                  "end": 8314
                                }
                              },
                              "loc": {
                                "start": 8302,
                                "end": 8314
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 8336,
                                      "end": 8353
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8333,
                                    "end": 8353
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8315,
                                "end": 8367
                              }
                            },
                            "loc": {
                              "start": 8295,
                              "end": 8367
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 8387,
                                  "end": 8394
                                }
                              },
                              "loc": {
                                "start": 8387,
                                "end": 8394
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 8416,
                                      "end": 8428
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8413,
                                    "end": 8428
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8395,
                                "end": 8442
                              }
                            },
                            "loc": {
                              "start": 8380,
                              "end": 8442
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Question",
                                "loc": {
                                  "start": 8462,
                                  "end": 8470
                                }
                              },
                              "loc": {
                                "start": 8462,
                                "end": 8470
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Question_list",
                                    "loc": {
                                      "start": 8492,
                                      "end": 8505
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8489,
                                    "end": 8505
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8471,
                                "end": 8519
                              }
                            },
                            "loc": {
                              "start": 8455,
                              "end": 8519
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 8539,
                                  "end": 8546
                                }
                              },
                              "loc": {
                                "start": 8539,
                                "end": 8546
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 8568,
                                      "end": 8580
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8565,
                                    "end": 8580
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8547,
                                "end": 8594
                              }
                            },
                            "loc": {
                              "start": 8532,
                              "end": 8594
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "SmartContract",
                                "loc": {
                                  "start": 8614,
                                  "end": 8627
                                }
                              },
                              "loc": {
                                "start": 8614,
                                "end": 8627
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "SmartContract_list",
                                    "loc": {
                                      "start": 8649,
                                      "end": 8667
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8646,
                                    "end": 8667
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8628,
                                "end": 8681
                              }
                            },
                            "loc": {
                              "start": 8607,
                              "end": 8681
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Standard",
                                "loc": {
                                  "start": 8701,
                                  "end": 8709
                                }
                              },
                              "loc": {
                                "start": 8701,
                                "end": 8709
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Standard_list",
                                    "loc": {
                                      "start": 8731,
                                      "end": 8744
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8728,
                                    "end": 8744
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8710,
                                "end": 8758
                              }
                            },
                            "loc": {
                              "start": 8694,
                              "end": 8758
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "User",
                                "loc": {
                                  "start": 8778,
                                  "end": 8782
                                }
                              },
                              "loc": {
                                "start": 8778,
                                "end": 8782
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "User_list",
                                    "loc": {
                                      "start": 8804,
                                      "end": 8813
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8801,
                                    "end": 8813
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8783,
                                "end": 8827
                              }
                            },
                            "loc": {
                              "start": 8771,
                              "end": 8827
                            }
                          }
                        ],
                        "loc": {
                          "start": 8145,
                          "end": 8837
                        }
                      },
                      "loc": {
                        "start": 8140,
                        "end": 8837
                      }
                    }
                  ],
                  "loc": {
                    "start": 8115,
                    "end": 8843
                  }
                },
                "loc": {
                  "start": 8109,
                  "end": 8843
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8848,
                    "end": 8856
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 8867,
                          "end": 8878
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8867,
                        "end": 8878
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8887,
                          "end": 8899
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8887,
                        "end": 8899
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8908,
                          "end": 8921
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8908,
                        "end": 8921
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 8930,
                          "end": 8951
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8930,
                        "end": 8951
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8960,
                          "end": 8976
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8960,
                        "end": 8976
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8985,
                          "end": 9002
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8985,
                        "end": 9002
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 9011,
                          "end": 9027
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9011,
                        "end": 9027
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorSmartContract",
                        "loc": {
                          "start": 9036,
                          "end": 9058
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9036,
                        "end": 9058
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 9067,
                          "end": 9084
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9067,
                        "end": 9084
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 9093,
                          "end": 9106
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9093,
                        "end": 9106
                      }
                    }
                  ],
                  "loc": {
                    "start": 8857,
                    "end": 9112
                  }
                },
                "loc": {
                  "start": 8848,
                  "end": 9112
                }
              }
            ],
            "loc": {
              "start": 8103,
              "end": 9116
            }
          },
          "loc": {
            "start": 8079,
            "end": 9116
          }
        }
      ],
      "loc": {
        "start": 8075,
        "end": 9118
      }
    },
    "loc": {
      "start": 8031,
      "end": 9118
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
